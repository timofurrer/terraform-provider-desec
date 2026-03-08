// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure RecordsDataSource satisfies the datasource interface.
var _ datasource.DataSource = (*recordsDataSource)(nil)

// newRecordsDataSource creates a new RecordsDataSource.
func newRecordsDataSource() datasource.DataSource {
	return &recordsDataSource{}
}

// recordsDataSource lists all RRsets within a deSEC domain.
type recordsDataSource struct {
	client *api.Client
}

// recordsDataSourceModel describes the data source data model.
type recordsDataSourceModel struct {
	Domain  types.String `tfsdk:"domain"`
	Subname types.String `tfsdk:"subname"`
	Type    types.String `tfsdk:"type"`
	Records types.List   `tfsdk:"records"`
}

func (d *recordsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_records"
}

func (d *recordsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the list of DNS Resource Record Sets (RRsets) within a deSEC domain, with optional filtering by subname or type.",

		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name to list records for.",
				Required:            true,
			},
			"subname": schema.StringAttribute{
				MarkdownDescription: "Filter records by this subname. Leave unset to return records for all subnames.",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Filter records by this record type (e.g. `A`, `AAAA`, `TXT`). Leave unset to return all types.",
				Optional:            true,
			},
			"records": schema.ListNestedAttribute{
				MarkdownDescription: "List of RRsets matching the filter criteria.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain": schema.StringAttribute{
							MarkdownDescription: "The domain name.",
							Computed:            true,
						},
						"subname": schema.StringAttribute{
							MarkdownDescription: "The subdomain part.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The full DNS name.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The DNS record type.",
							Computed:            true,
						},
						"ttl": schema.Int64Attribute{
							MarkdownDescription: "The TTL in seconds.",
							Computed:            true,
						},
						"records": schema.SetAttribute{
							MarkdownDescription: "The set of record content strings.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"created": schema.StringAttribute{
							MarkdownDescription: "Timestamp of RRset creation.",
							Computed:            true,
						},
						"touched": schema.StringAttribute{
							MarkdownDescription: "Timestamp of last touch.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *recordsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *recordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data recordsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subname := ""
	if !data.Subname.IsNull() && !data.Subname.IsUnknown() {
		subname = data.Subname.ValueString()
	}
	rrtype := ""
	if !data.Type.IsNull() && !data.Type.IsUnknown() {
		rrtype = data.Type.ValueString()
	}

	rrsets, err := d.client.ListRRsets(ctx, data.Domain.ValueString(), subname, rrtype)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Records",
			fmt.Sprintf("Unable to list records for domain %q: %s", data.Domain.ValueString(), err))
		return
	}

	// Build the nested attribute types.
	rrsetAttrTypes := map[string]attr.Type{
		"domain":  types.StringType,
		"subname": types.StringType,
		"name":    types.StringType,
		"type":    types.StringType,
		"ttl":     types.Int64Type,
		"records": types.SetType{ElemType: types.StringType},
		"created": types.StringType,
		"touched": types.StringType,
	}
	rrsetObjType := types.ObjectType{AttrTypes: rrsetAttrTypes}

	rrsetObjs := make([]attr.Value, 0, len(rrsets))
	for _, rs := range rrsets {
		recordSet, setDiags := types.SetValueFrom(ctx, types.StringType, rs.Records)
		resp.Diagnostics.Append(setDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		obj, objDiags := types.ObjectValue(rrsetAttrTypes, map[string]attr.Value{
			"domain":  types.StringValue(rs.Domain),
			"subname": types.StringValue(normalizeSubname(rs.Subname)),
			"name":    types.StringValue(rs.Name),
			"type":    types.StringValue(rs.Type),
			"ttl":     types.Int64Value(int64(rs.TTL)),
			"records": recordSet,
			"created": types.StringValue(rs.Created),
			"touched": types.StringValue(rs.Touched),
		})
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		rrsetObjs = append(rrsetObjs, obj)
	}

	rrsetsList, listDiags := types.ListValue(rrsetObjType, rrsetObjs)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Records = rrsetsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
