// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure RecordResource fully satisfies framework interfaces.
var _ resource.Resource = (*recordResource)(nil)
var _ resource.ResourceWithImportState = (*recordResource)(nil)
var _ resource.ResourceWithIdentity = (*recordResource)(nil)

// subnameStringType is a custom string type for the subname attribute.
// It treats "" and "@" as semantically equal, both representing the zone apex.
type subnameStringType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = subnameStringType{}

func (t subnameStringType) Equal(o attr.Type) bool {
	other, ok := o.(subnameStringType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t subnameStringType) String() string {
	return "subnameStringType"
}

func (t subnameStringType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return subnameStringValue{StringValue: in}, nil
}

func (t subnameStringType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrVal, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	sv, ok := attrVal.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", attrVal)
	}
	val, diags := t.ValueFromString(ctx, sv)
	if diags.HasError() {
		return nil, fmt.Errorf("converting StringValue to subnameStringValue: %v", diags)
	}
	return val, nil
}

func (t subnameStringType) ValueType(_ context.Context) attr.Value {
	return subnameStringValue{}
}

// subnameStringValue is the value type for subnameStringType.
// It implements semantic equality so that "" and "@" are treated as the same value.
type subnameStringValue struct {
	basetypes.StringValue
}

var _ basetypes.StringValuable = subnameStringValue{}
var _ basetypes.StringValuableWithSemanticEquals = subnameStringValue{}

func (v subnameStringValue) Equal(o attr.Value) bool {
	other, ok := o.(subnameStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v subnameStringValue) Type(_ context.Context) attr.Type {
	return subnameStringType{}
}

// StringSemanticEquals returns true when both values represent the zone apex,
// i.e. one is "" and the other is "@". This prevents a persistent plan diff
// when the user writes subname = "" and the provider stores "@" in state.
func (v subnameStringValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	newVal, ok := newValuable.(subnameStringValue)
	if !ok {
		return false, nil
	}
	prior := v.ValueString()
	next := newVal.ValueString()
	isApex := func(s string) bool { return s == "" || s == "@" }
	return isApex(prior) && isApex(next), nil
}

// recordIdentityModel describes the identity of a record resource.
type recordIdentityModel struct {
	Domain  types.String `tfsdk:"domain"`
	Subname types.String `tfsdk:"subname"`
	Type    types.String `tfsdk:"type"`
}

func (r *recordResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"domain": identityschema.StringAttribute{
				Description:       "The domain name this record belongs to.",
				RequiredForImport: true,
			},
			"subname": identityschema.StringAttribute{
				Description:       "The subdomain part of the record name.",
				RequiredForImport: true,
			},
			"type": identityschema.StringAttribute{
				Description:       "The DNS record type (e.g. A, AAAA, TXT).",
				RequiredForImport: true,
			},
		},
	}
}

// newRecordResource creates a new RecordResource.
func newRecordResource() resource.Resource {
	return &recordResource{}
}

// recordResource manages a deSEC RRset (DNS resource record set).
type recordResource struct {
	client *api.Client
}

// recordResourceModel describes the resource data model.
type recordResourceModel struct {
	ID      types.String       `tfsdk:"id"`
	Domain  types.String       `tfsdk:"domain"`
	Subname subnameStringValue `tfsdk:"subname"`
	Type    types.String       `tfsdk:"type"`
	TTL     types.Int64        `tfsdk:"ttl"`
	Records types.Set          `tfsdk:"records"`
	Created types.String       `tfsdk:"created"`
	Touched types.String       `tfsdk:"touched"`
}

func (r *recordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_record"
}

func (r *recordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a deSEC DNS Resource Record Set (RRset). An RRset is the set of all DNS records of a given type for a given name within a domain.\n\n" +
			"Use `@` as the `subname` to manage records at the zone apex (root of the domain).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The resource identifier in the form `domain/subname/type`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name that this record belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subname": schema.StringAttribute{
				CustomType:          subnameStringType{},
				MarkdownDescription: "The subdomain part of the record name. Use `@` or `\"\"` for the zone apex (root of the domain).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The DNS record type (e.g. `A`, `AAAA`, `CNAME`, `MX`, `TXT`). Must be uppercase.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Z][A-Z0-9]*$`), "must be an uppercase DNS record type (e.g. A, AAAA, MX, TXT)"),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "The TTL (time-to-live) in seconds. Must be at least the domain's `minimum_ttl`.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"records": schema.SetAttribute{
				MarkdownDescription: "The set of record content strings. The format depends on the record type. For example, `A` records contain IPv4 addresses, `MX` records contain `priority hostname.` pairs.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Timestamp of RRset creation in ISO 8601 format.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"touched": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the RRset was last touched.",
				Computed:            true,
			},
		},
	}
}

func (r *recordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *recordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data recordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, diags := recordsFromSet(ctx, data.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rrset, err := r.client.CreateRRset(ctx,
		data.Domain.ValueString(),
		api.CreateRRsetOptions{
			Subname: data.Subname.ValueString(),
			Type:    data.Type.ValueString(),
			TTL:     int(data.TTL.ValueInt64()),
			Records: records,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Record",
			fmt.Sprintf("Unable to create record %s/%s/%s: %s",
				data.Domain.ValueString(), data.Subname.ValueString(), data.Type.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(rrsetToModel(ctx, rrset, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, recordIdentityModel{
		Domain:  data.Domain,
		Subname: types.StringValue(data.Subname.ValueString()),
		Type:    data.Type,
	})...)
}

func (r *recordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data recordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rrset, err := r.client.GetRRset(ctx,
		data.Domain.ValueString(),
		data.Subname.ValueString(),
		data.Type.ValueString(),
	)
	if err != nil {
		if api.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Record",
			fmt.Sprintf("Unable to read record %s/%s/%s: %s",
				data.Domain.ValueString(), data.Subname.ValueString(), data.Type.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(rrsetToModel(ctx, rrset, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, recordIdentityModel{
		Domain:  data.Domain,
		Subname: types.StringValue(data.Subname.ValueString()),
		Type:    data.Type,
	})...)
}

func (r *recordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data recordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, diags := recordsFromSet(ctx, data.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rrset, err := r.client.UpdateRRset(ctx,
		data.Domain.ValueString(),
		data.Subname.ValueString(),
		data.Type.ValueString(),
		api.UpdateRRsetOptions{
			TTL:     int(data.TTL.ValueInt64()),
			Records: records,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Record",
			fmt.Sprintf("Unable to update record %s/%s/%s: %s",
				data.Domain.ValueString(), data.Subname.ValueString(), data.Type.ValueString(), err))
		return
	}

	resp.Diagnostics.Append(rrsetToModel(ctx, rrset, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *recordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data recordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRRset(ctx,
		data.Domain.ValueString(),
		data.Subname.ValueString(),
		data.Type.ValueString(),
	); err != nil {
		if api.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Record",
			fmt.Sprintf("Unable to delete record %s/%s/%s: %s",
				data.Domain.ValueString(), data.Subname.ValueString(), data.Type.ValueString(), err))
	}
}

func (r *recordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		// Import format: "domain/subname/type".
		// The canonical apex form is "example.com/@/A"; the empty-segment form
		// "example.com//A" is also accepted. Both are normalised to "@" here so
		// that the intermediate import state is consistent with post-Read state.
		parts := strings.SplitN(req.ID, "/", 3)
		if len(parts) != 3 {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Expected import ID in the format 'domain/subname/type', got %q", req.ID),
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subname"), normalizeSubname(parts[1]))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[2])...)
		return
	}
	var identity recordIdentityModel
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), identity.Domain)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subname"), identity.Subname)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), identity.Type)...)
}

// normalizeSubname converts an empty subname (zone apex as returned by the API)
// to "@" for consistent representation in Terraform state and import IDs.
func normalizeSubname(subname string) string {
	if subname == "" {
		return "@"
	}
	return subname
}

// rrsetToModel converts an api.RRset into a RecordResourceModel.
func rrsetToModel(ctx context.Context, rs *api.RRset, m *recordResourceModel) diag.Diagnostics {
	subname := normalizeSubname(rs.Subname)
	m.ID = types.StringValue(rs.Domain + "/" + subname + "/" + rs.Type)
	m.Domain = types.StringValue(rs.Domain)
	m.Subname = subnameStringValue{StringValue: basetypes.NewStringValue(subname)}
	m.Type = types.StringValue(rs.Type)
	m.TTL = types.Int64Value(int64(rs.TTL))
	m.Created = types.StringValue(rs.Created)
	m.Touched = types.StringValue(rs.Touched)

	recordVals := make([]string, len(rs.Records))
	copy(recordVals, rs.Records)

	setVal, diags := types.SetValueFrom(ctx, types.StringType, recordVals)
	if diags.HasError() {
		return diags
	}
	m.Records = setVal
	return diags
}

// recordsFromSet extracts the string slice from a types.Set of strings.
func recordsFromSet(ctx context.Context, s types.Set) ([]string, diag.Diagnostics) {
	var records []string
	diags := s.ElementsAs(ctx, &records, false)
	return records, diags
}
