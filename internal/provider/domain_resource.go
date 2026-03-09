// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure DomainResource fully satisfies framework interfaces.
var _ resource.Resource = (*domainResource)(nil)
var _ resource.ResourceWithImportState = (*domainResource)(nil)
var _ resource.ResourceWithIdentity = (*domainResource)(nil)

// domainIdentityModel describes the identity of a domain resource.
type domainIdentityModel struct {
	Name types.String `tfsdk:"name"`
}

func (r *domainResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"name": identityschema.StringAttribute{
				Description:       "The domain name.",
				RequiredForImport: true,
			},
		},
	}
}

// newDomainResource creates a new DomainResource.
func newDomainResource() resource.Resource {
	return &domainResource{}
}

// domainResource manages a deSEC domain (DNS zone).
type domainResource struct {
	client *api.Client
}

// domainResourceModel describes the resource data model.
type domainResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	MinimumTTL types.Int64  `tfsdk:"minimum_ttl"`
	Created    types.String `tfsdk:"created"`
	Published  types.String `tfsdk:"published"`
	Touched    types.String `tfsdk:"touched"`
	Keys       types.List   `tfsdk:"keys"`
}

// keyAttrTypes defines the attribute types for the keys list elements.
var keyAttrTypes = map[string]attr.Type{
	"dnskey":  types.StringType,
	"ds":      types.ListType{ElemType: types.StringType},
	"managed": types.BoolType,
}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a deSEC DNS zone (domain). Domain names are immutable after creation.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain name, used as the resource identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The domain name (e.g. `example.com`). Must be unique and is immutable after creation.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"minimum_ttl": schema.Int64Attribute{
				MarkdownDescription: "The minimum TTL (in seconds) that can be used for RRsets in this domain. Set automatically by the server.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Timestamp of domain creation in ISO 8601 format.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"published": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the domain's DNS records were last published.",
				Computed:            true,
			},
			"touched": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the domain's DNS records were last touched.",
				Computed:            true,
			},
			"keys": schema.ListNestedAttribute{
				MarkdownDescription: "DNSSEC public key information for the domain.",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"dnskey": schema.StringAttribute{
							MarkdownDescription: "The DNSKEY record content.",
							Computed:            true,
						},
						"ds": schema.ListAttribute{
							MarkdownDescription: "The DS record contents (for delegation).",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"managed": schema.BoolAttribute{
							MarkdownDescription: "Whether this key is managed by deSEC.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data domainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.CreateDomain(ctx, api.CreateDomainOptions{Name: data.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Domain", fmt.Sprintf("Unable to create domain %q: %s", data.Name.ValueString(), err))
		return
	}

	diags := domainToModel(domain, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, domainIdentityModel{
		Name: types.StringValue(domain.Name),
	})...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, data.Name.ValueString())
	if err != nil {
		if api.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Domain", fmt.Sprintf("Unable to read domain %q: %s", data.Name.ValueString(), err))
		return
	}

	diags := domainToModel(domain, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, domainIdentityModel{
		Name: types.StringValue(domain.Name),
	})...)
}

// Update is not implemented because all domain fields are either read-only or
// immutable (name is write-once). If name changes, Terraform will trigger a
// destroy+create via RequiresReplace.
func (r *domainResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data domainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteDomain(ctx, data.Name.ValueString()); err != nil {
		if api.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Domain", fmt.Sprintf("Unable to delete domain %q: %s", data.Name.ValueString(), err))
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("name"), path.Root("name"), req, resp)
}

// domainToModel converts an api.Domain into a DomainResourceModel, filling in
// all computed fields. Returns diagnostics for any conversion errors.
func domainToModel(d *api.Domain, m *domainResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(d.Name)
	m.Name = types.StringValue(d.Name)
	m.MinimumTTL = types.Int64Value(int64(d.MinimumTTL))
	m.Created = types.StringValue(d.Created)
	m.Published = types.StringValue(d.Published)
	m.Touched = types.StringValue(d.Touched)

	// Convert keys.
	keyObjType := types.ObjectType{AttrTypes: keyAttrTypes}
	keyObjs := make([]attr.Value, 0, len(d.Keys))
	for _, k := range d.Keys {
		dsVals := make([]attr.Value, 0, len(k.DS))
		for _, ds := range k.DS {
			dsVals = append(dsVals, types.StringValue(ds))
		}
		dsList, listDiags := types.ListValue(types.StringType, dsVals)
		diags.Append(listDiags...)
		if diags.HasError() {
			return diags
		}

		keyObj, objDiags := types.ObjectValue(keyAttrTypes, map[string]attr.Value{
			"dnskey":  types.StringValue(k.DNSKey),
			"ds":      dsList,
			"managed": types.BoolValue(k.Managed),
		})
		diags.Append(objDiags...)
		if diags.HasError() {
			return diags
		}
		keyObjs = append(keyObjs, keyObj)
	}

	keysList, listDiags := types.ListValue(keyObjType, keyObjs)
	diags.Append(listDiags...)
	if diags.HasError() {
		return diags
	}
	m.Keys = keysList

	return diags
}
