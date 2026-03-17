// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

var _ resource.Resource = (*recordsResource)(nil)
var _ resource.ResourceWithImportState = (*recordsResource)(nil)
var _ resource.ResourceWithValidateConfig = (*recordsResource)(nil)

var autoManagedRRTypes = map[uint16]bool{
	dns.TypeSOA:        true,
	dns.TypeRRSIG:      true,
	dns.TypeNSEC:       true,
	dns.TypeNSEC3:      true,
	dns.TypeNSEC3PARAM: true,
	dns.TypeCDNSKEY:    true,
	dns.TypeCDS:        true,
}

var recordsRRsetObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"subname": subnameStringType{},
		"type":    types.StringType,
		"ttl":     types.Int64Type,
		"records": types.SetType{ElemType: types.StringType},
	},
}

func newRecordsResource() resource.Resource {
	return &recordsResource{}
}

type recordsResource struct {
	client *api.Client
}

type recordsResourceModel struct {
	Domain    types.String `tfsdk:"domain"`
	Exclusive types.Bool   `tfsdk:"exclusive"`
	Zonefile  types.String `tfsdk:"zonefile"`
	Records   types.Set    `tfsdk:"records"`
}

type recordsRRsetModel struct {
	Subname subnameStringValue `tfsdk:"subname"`
	Type    types.String       `tfsdk:"type"`
	TTL     types.Int64        `tfsdk:"ttl"`
	Records types.Set          `tfsdk:"records"`
}

func (r *recordsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_records"
}

func (r *recordsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a set of deSEC DNS records via the bulk RRset API.\n\n" +
			"Records can be specified in one of two ways (mutually exclusive):\n\n" +
			"- **Mode A (`zonefile`)**: Provide a BIND-format zone file string. The `records` attribute is computed.\n" +
			"- **Mode B (`records`)**: Provide a structured set of RRset objects. The `zonefile` attribute is computed.\n\n" +
			"This resource **co-exists** with `desec_record` resources. Only the RRsets explicitly declared " +
			"are owned and managed; other records in the domain are not touched.\n\n" +
			"The following record types are silently ignored because they are managed automatically by deSEC: " +
			"`SOA`, `RRSIG`, `NSEC`, `NSEC3`, `NSEC3PARAM`, `CDNSKEY`, `CDS`, and apex `NS` records.",

		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name to manage records for. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"exclusive": schema.BoolAttribute{
				MarkdownDescription: "When `true`, only the declared RRsets may exist on the domain " +
					"(excluding automatically managed types such as SOA, RRSIG, NSEC*, CDNSKEY, CDS, " +
					"and apex NS). Any other RRsets found on the domain are deleted.\n\n" +
					"When `false` (the default), this resource co-exists with other records on the domain " +
					"and only manages the RRsets explicitly declared in the configuration.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"zonefile": schema.StringAttribute{
				MarkdownDescription: "Zone file content in RFC 1035 / BIND format.\n\n" +
					"Mutually exclusive with `records`. When set, the `records` attribute is computed from the " +
					"parsed zone file. When `records` is set instead, this attribute is computed as a canonical " +
					"reconstruction of the live records.\n\n" +
					"Format and ordering differences (comments, whitespace, record ordering) are suppressed: " +
					"a plan will only show a diff when the set of records or their values actually change.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					recordsZonefileModifier{},
				},
			},
			"records": schema.SetNestedAttribute{
				MarkdownDescription: "Structured set of RRset objects.\n\n" +
					"Mutually exclusive with `zonefile`. When set, the `zonefile` attribute is computed. " +
					"When `zonefile` is set instead, this attribute is computed from the parsed zone file.\n\n" +
					"Each element represents one RRset (a unique `(subname, type)` pair).",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Set{
					recordsSetModifier{},
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"subname": schema.StringAttribute{
							CustomType:          subnameStringType{},
							MarkdownDescription: "The subdomain component. Use `\"\"` or `\"@\"` for the zone apex.",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The DNS record type (e.g. `A`, `AAAA`, `MX`, `TXT`). Must be uppercase.",
							Required:            true,
						},
						"ttl": schema.Int64Attribute{
							MarkdownDescription: "Time-to-live in seconds.",
							Required:            true,
						},
						"records": schema.SetAttribute{
							MarkdownDescription: "Record values in presentation format (RDATA only).",
							Required:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *recordsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data recordsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zonefileSet := !data.Zonefile.IsNull() && !data.Zonefile.IsUnknown()
	recordsSet := !data.Records.IsNull() && !data.Records.IsUnknown()

	if zonefileSet && recordsSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("zonefile"),
			"Conflicting Configuration",
			"Only one of \"zonefile\" or \"records\" may be specified, not both.",
		)
		return
	}

	if !zonefileSet && !recordsSet {
		resp.Diagnostics.AddError(
			"Missing Configuration",
			"Exactly one of \"zonefile\" or \"records\" must be specified.",
		)
		return
	}
}

func (r *recordsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ---- CRUD ----

func (r *recordsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data recordsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()

	rrsets, diags := r.resolveRRsets(ctx, data, domain)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries := rrsetsToPutEntries(rrsets)

	if data.Exclusive.ValueBool() {
		allRRsets, err := r.client.ListRRsets(ctx, domain, api.ListRRsetsOptions{})
		if err != nil && !api.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Listing Records",
				fmt.Sprintf("Unable to list records for domain %q: %s", domain, err))
			return
		}
		entries = append(entries, deletionEntriesForExtras(allRRsets, rrsets)...)
	}

	returned, err := r.client.BulkPutRRsets(ctx, domain, entries)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Records",
			fmt.Sprintf("Unable to create records for domain %q: %s", domain, err))
		return
	}

	sortRRsets(returned)
	resp.Diagnostics.Append(r.setStateAfterWrite(ctx, &data, domain, returned)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *recordsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data recordsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := data.Domain.ValueString()

	allRRsets, err := r.client.ListRRsets(ctx, domain, api.ListRRsetsOptions{})
	if err != nil {
		if api.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Records",
			fmt.Sprintf("Unable to list records for domain %q: %s", domain, err))
		return
	}

	var managed []api.RRset

	importCase := data.Records.IsNull() || data.Records.IsUnknown() || len(data.Records.Elements()) == 0
	if importCase || data.Exclusive.ValueBool() {
		for _, rs := range allRRsets {
			rrtype := dns.StringToType[rs.Type]
			if autoManagedRRTypes[rrtype] {
				continue
			}
			if isApexNS(rs.Subname, rs.Type) {
				continue
			}
			managed = append(managed, rs)
		}
	} else {
		stateRRsets, diags := recordsSetToAPIRRsets(ctx, data.Records)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		type rrKey struct{ subname, rrtype string }
		owned := make(map[rrKey]bool, len(stateRRsets))
		for _, rs := range stateRRsets {
			owned[rrKey{rs.Subname, rs.Type}] = true
		}
		for _, rs := range allRRsets {
			if owned[rrKey{rs.Subname, rs.Type}] {
				managed = append(managed, rs)
			}
		}
	}

	sortRRsets(managed)

	recordsSet, diags := apiRRsetsToSet(ctx, managed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Records = recordsSet
	data.Zonefile = types.StringValue(rrsetToZonefile(domain, managed))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *recordsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan recordsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state recordsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain := plan.Domain.ValueString()

	newRRsets, diags := r.resolveRRsets(ctx, plan, domain)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldRRsets, diags := recordsSetToAPIRRsets(ctx, state.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	type rrKey struct{ subname, rrtype string }
	newSet := make(map[rrKey]bool, len(newRRsets))
	for _, rs := range newRRsets {
		newSet[rrKey{rs.Subname, rs.Type}] = true
	}

	entries := rrsetsToPutEntries(newRRsets)
	for _, rs := range oldRRsets {
		if !newSet[rrKey{rs.Subname, rs.Type}] {
			entries = append(entries, api.BulkPutRRsetEntry{
				Subname: rs.Subname,
				Type:    rs.Type,
				TTL:     rs.TTL,
				Records: []string{},
			})
		}
	}

	if plan.Exclusive.ValueBool() {
		allRRsets, err := r.client.ListRRsets(ctx, domain, api.ListRRsetsOptions{})
		if err != nil && !api.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Listing Records",
				fmt.Sprintf("Unable to list records for domain %q: %s", domain, err))
			return
		}
		entries = append(entries, deletionEntriesForExtras(allRRsets, newRRsets)...)
	}

	returned, err := r.client.BulkPutRRsets(ctx, domain, entries)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Records",
			fmt.Sprintf("Unable to update records for domain %q: %s", domain, err))
		return
	}

	sortRRsets(returned)
	resp.Diagnostics.Append(r.setStateAfterWrite(ctx, &plan, domain, returned)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recordsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data recordsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateRRsets, diags := recordsSetToAPIRRsets(ctx, data.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(stateRRsets) == 0 {
		return
	}

	entries := make([]api.BulkPutRRsetEntry, len(stateRRsets))
	for i, rs := range stateRRsets {
		entries[i] = api.BulkPutRRsetEntry{
			Subname: rs.Subname,
			Type:    rs.Type,
			TTL:     rs.TTL,
			Records: []string{},
		}
	}

	if _, err := r.client.BulkPutRRsets(ctx, data.Domain.ValueString(), entries); err != nil {
		if api.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Records",
			fmt.Sprintf("Unable to delete records for domain %q: %s", data.Domain.ValueString(), err))
	}
}

func (r *recordsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

// ---- Helpers ----

func (r *recordsResource) resolveRRsets(ctx context.Context, data recordsResourceModel, domain string) ([]api.RRset, diag.Diagnostics) {
	if !data.Zonefile.IsNull() && !data.Zonefile.IsUnknown() {
		rrsets, err := parseZonefile(data.Zonefile.ValueString(), domain)
		if err != nil {
			var diags diag.Diagnostics
			diags.AddError("Invalid Zone File", fmt.Sprintf("Unable to parse zone file: %s", err))
			return nil, diags
		}
		return rrsets, nil
	}

	if !data.Records.IsNull() && !data.Records.IsUnknown() {
		return recordsSetToAPIRRsets(ctx, data.Records)
	}

	var diags diag.Diagnostics
	diags.AddError("Missing Configuration", "Either \"zonefile\" or \"records\" must be specified.")
	return nil, diags
}

func (r *recordsResource) setStateAfterWrite(ctx context.Context, data *recordsResourceModel, domain string, returned []api.RRset) diag.Diagnostics {
	recordsSet, diags := apiRRsetsToSet(ctx, returned)
	if diags.HasError() {
		return diags
	}

	isZonefileMode := !data.Zonefile.IsNull() && !data.Zonefile.IsUnknown() && data.Zonefile.ValueString() != ""

	if isZonefileMode {
		data.Records = recordsSet
	} else {
		data.Zonefile = types.StringValue(rrsetToZonefile(domain, returned))
	}

	return diags
}

func isApexNS(subname, rrtype string) bool {
	return subname == "" && rrtype == "NS"
}

func parseZonefile(zonefile, domain string) ([]api.RRset, error) {
	origin := dnsutil.Fqdn(domain)
	zp := dns.NewZoneParser(strings.NewReader(zonefile), origin, "")

	type key struct{ subname, rrtype string }
	type group struct {
		ttl     int
		records []string
	}
	groups := map[key]*group{}

	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		rrtype := dns.RRToType(rr)
		if autoManagedRRTypes[rrtype] {
			continue
		}

		subname, inDomain := extractSubname(rr.Header().Name, origin)
		if !inDomain {
			continue
		}

		rrtypeStr := dns.TypeToString[rrtype]
		if isApexNS(subname, rrtypeStr) {
			continue
		}

		parts := strings.SplitN(rr.String(), "\t", 5)
		if len(parts) < 5 {
			continue
		}
		rdata := parts[4]

		k := key{subname, rrtypeStr}
		if groups[k] == nil {
			groups[k] = &group{ttl: int(rr.Header().TTL)}
		}
		groups[k].records = append(groups[k].records, rdata)
	}
	if err := zp.Err(); err != nil {
		return nil, fmt.Errorf("parsing zone file: %w", err)
	}

	result := make([]api.RRset, 0, len(groups))
	for k, g := range groups {
		recs := make([]string, len(g.records))
		copy(recs, g.records)
		sort.Strings(recs)
		result = append(result, api.RRset{
			Subname: k.subname,
			Type:    k.rrtype,
			TTL:     g.ttl,
			Records: recs,
		})
	}
	sortRRsets(result)
	return result, nil
}

func extractSubname(fqdn, origin string) (string, bool) {
	if fqdn == origin {
		return "", true
	}
	suffix := "." + origin
	if before, ok := strings.CutSuffix(fqdn, suffix); ok {
		return before, true
	}
	return "", false
}

func rrsetToZonefile(domain string, rrsets []api.RRset) string {
	sorted := make([]api.RRset, len(rrsets))
	copy(sorted, rrsets)
	sortRRsets(sorted)

	var sb strings.Builder
	for _, rs := range sorted {
		var owner string
		if rs.Subname == "" {
			owner = dnsutil.Fqdn(domain)
		} else {
			owner = rs.Subname + "." + dnsutil.Fqdn(domain)
		}
		recs := make([]string, len(rs.Records))
		copy(recs, rs.Records)
		sort.Strings(recs)
		for _, rec := range recs {
			fmt.Fprintf(&sb, "%s\t%d\tIN\t%s\t%s\n", owner, rs.TTL, rs.Type, rec)
		}
	}
	return sb.String()
}

func sortRRsets(rrsets []api.RRset) {
	sort.Slice(rrsets, func(i, j int) bool {
		if rrsets[i].Subname != rrsets[j].Subname {
			return rrsets[i].Subname < rrsets[j].Subname
		}
		return rrsets[i].Type < rrsets[j].Type
	})
}

func rrsetSetsEqual(a, b []api.RRset) bool {
	if len(a) != len(b) {
		return false
	}

	type rrKey struct{ subname, rrtype string }
	type rrVal struct {
		ttl     int
		records string
	}

	toMap := func(rrsets []api.RRset) map[rrKey]rrVal {
		m := make(map[rrKey]rrVal, len(rrsets))
		for _, rs := range rrsets {
			recs := make([]string, len(rs.Records))
			copy(recs, rs.Records)
			sort.Strings(recs)
			m[rrKey{rs.Subname, rs.Type}] = rrVal{rs.TTL, strings.Join(recs, "\x00")}
		}
		return m
	}

	ma, mb := toMap(a), toMap(b)
	if len(ma) != len(mb) {
		return false
	}
	for k, va := range ma {
		vb, ok := mb[k]
		if !ok || va != vb {
			return false
		}
	}
	return true
}

func rrsetsToPutEntries(rrsets []api.RRset) []api.BulkPutRRsetEntry {
	entries := make([]api.BulkPutRRsetEntry, len(rrsets))
	for i, rs := range rrsets {
		entries[i] = api.BulkPutRRsetEntry{
			Subname: rs.Subname,
			Type:    rs.Type,
			TTL:     rs.TTL,
			Records: rs.Records,
		}
	}
	return entries
}

func deletionEntriesForExtras(allRRsets []api.RRset, configured []api.RRset) []api.BulkPutRRsetEntry {
	type rrKey struct{ subname, rrtype string }
	wanted := make(map[rrKey]bool, len(configured))
	for _, rs := range configured {
		wanted[rrKey{rs.Subname, rs.Type}] = true
	}

	var entries []api.BulkPutRRsetEntry
	for _, rs := range allRRsets {
		rrtype := dns.StringToType[rs.Type]
		if autoManagedRRTypes[rrtype] {
			continue
		}
		if isApexNS(rs.Subname, rs.Type) {
			continue
		}
		if wanted[rrKey{rs.Subname, rs.Type}] {
			continue
		}
		entries = append(entries, api.BulkPutRRsetEntry{
			Subname: rs.Subname,
			Type:    rs.Type,
			TTL:     rs.TTL,
			Records: []string{},
		})
	}
	return entries
}

func apiRRsetsToSet(ctx context.Context, rrsets []api.RRset) (types.Set, diag.Diagnostics) {
	objs := make([]attr.Value, len(rrsets))
	for i, rs := range rrsets {
		recs := make([]string, len(rs.Records))
		copy(recs, rs.Records)
		sort.Strings(recs)

		recsSet, diags := types.SetValueFrom(ctx, types.StringType, recs)
		if diags.HasError() {
			return types.SetNull(recordsRRsetObjectType), diags
		}

		obj, diags := types.ObjectValue(recordsRRsetObjectType.AttrTypes, map[string]attr.Value{
			"subname": subnameStringValue{StringValue: types.StringValue(rs.Subname)},
			"type":    types.StringValue(rs.Type),
			"ttl":     types.Int64Value(int64(rs.TTL)),
			"records": recsSet,
		})
		if diags.HasError() {
			return types.SetNull(recordsRRsetObjectType), diags
		}
		objs[i] = obj
	}

	return types.SetValue(recordsRRsetObjectType, objs)
}

func recordsSetToAPIRRsets(ctx context.Context, set types.Set) ([]api.RRset, diag.Diagnostics) {
	var models []recordsRRsetModel
	diags := set.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}

	rrsets := make([]api.RRset, len(models))
	for i, m := range models {
		var recs []string
		diags = m.Records.ElementsAs(ctx, &recs, false)
		if diags.HasError() {
			return nil, diags
		}
		subname := m.Subname.ValueString()
		if subname == "@" {
			subname = ""
		}
		rrsets[i] = api.RRset{
			Subname: subname,
			Type:    m.Type.ValueString(),
			TTL:     int(m.TTL.ValueInt64()),
			Records: recs,
		}
	}
	return rrsets, nil
}

// ---- Plan modifiers ----

type recordsZonefileModifier struct{}

func (recordsZonefileModifier) Description(_ context.Context) string {
	return "Handles semantic equality for the zonefile attribute across both input and computed modes."
}

func (m recordsZonefileModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (recordsZonefileModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	var stateData recordsResourceModel
	if diags := req.State.Get(ctx, &stateData); diags.HasError() {
		return
	}
	if stateData.Records.IsNull() || stateData.Records.IsUnknown() {
		return
	}

	stateRRsets, diags := recordsSetToAPIRRsets(ctx, stateData.Records)
	if diags.HasError() {
		return
	}

	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		if req.ConfigValue.Equal(req.StateValue) {
			return
		}
		newRRsets, err := parseZonefile(req.ConfigValue.ValueString(), stateData.Domain.ValueString())
		if err != nil {
			return
		}
		if rrsetSetsEqual(newRRsets, stateRRsets) {
			resp.PlanValue = req.StateValue
		}
		return
	}

	if !resp.PlanValue.IsUnknown() {
		return
	}

	var planData recordsResourceModel
	if diags := req.Plan.Get(ctx, &planData); diags.HasError() {
		return
	}
	if planData.Records.IsNull() || planData.Records.IsUnknown() {
		return
	}

	newRRsets, diags := recordsSetToAPIRRsets(ctx, planData.Records)
	if diags.HasError() {
		return
	}

	if rrsetSetsEqual(newRRsets, stateRRsets) {
		resp.PlanValue = req.StateValue
	}
}

type recordsSetModifier struct{}

func (recordsSetModifier) Description(_ context.Context) string {
	return "Handles semantic equality for the records attribute across both input and computed modes."
}

func (m recordsSetModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (recordsSetModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	if !resp.PlanValue.IsUnknown() {
		return
	}

	var stateData recordsResourceModel
	if diags := req.State.Get(ctx, &stateData); diags.HasError() {
		return
	}
	if stateData.Records.IsNull() || stateData.Records.IsUnknown() {
		return
	}
	if stateData.Domain.IsNull() || stateData.Domain.IsUnknown() {
		return
	}

	stateRRsets, diags := recordsSetToAPIRRsets(ctx, stateData.Records)
	if diags.HasError() {
		return
	}

	var planData recordsResourceModel
	if diags := req.Plan.Get(ctx, &planData); diags.HasError() {
		return
	}

	if !planData.Zonefile.IsNull() && !planData.Zonefile.IsUnknown() {
		newRRsets, err := parseZonefile(planData.Zonefile.ValueString(), stateData.Domain.ValueString())
		if err != nil {
			return
		}
		if rrsetSetsEqual(newRRsets, stateRRsets) {
			resp.PlanValue = req.StateValue
		}
	}
}
