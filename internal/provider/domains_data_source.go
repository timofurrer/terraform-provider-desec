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

// Ensure DomainsDataSource satisfies the datasource interface.
var _ datasource.DataSource = (*domainsDataSource)(nil)

// newDomainsDataSource creates a new DomainsDataSource.
func newDomainsDataSource() datasource.DataSource {
	return &domainsDataSource{}
}

// domainsDataSource lists all deSEC domains, with an optional qname ownership filter.
type domainsDataSource struct {
	client *api.Client
}

// domainsDataSourceModel describes the data source data model.
type domainsDataSourceModel struct {
	OwnsQname types.String `tfsdk:"owns_qname"`
	Domains   types.List   `tfsdk:"domains"`
}

func (d *domainsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

func (d *domainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the list of all deSEC DNS zones owned by the account.",

		Attributes: map[string]schema.Attribute{
			"owns_qname": schema.StringAttribute{
				MarkdownDescription: "Filter to return only the domain responsible for this DNS query name (at most one result).",
				Optional:            true,
			},
			"domains": schema.ListNestedAttribute{
				MarkdownDescription: "List of domains.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The domain name.",
							Computed:            true,
						},
						"minimum_ttl": schema.Int64Attribute{
							MarkdownDescription: "The minimum TTL for RRsets in this domain.",
							Computed:            true,
						},
						"created": schema.StringAttribute{
							MarkdownDescription: "Timestamp of domain creation.",
							Computed:            true,
						},
						"published": schema.StringAttribute{
							MarkdownDescription: "Timestamp of last DNS publication.",
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

func (d *domainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *domainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ownsQname := ""
	if !data.OwnsQname.IsNull() && !data.OwnsQname.IsUnknown() {
		ownsQname = data.OwnsQname.ValueString()
	}

	domains, err := d.client.ListDomains(ctx, ownsQname)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Domains", fmt.Sprintf("Unable to list domains: %s", err))
		return
	}

	domainObjAttrTypes := map[string]attr.Type{
		"name":        types.StringType,
		"minimum_ttl": types.Int64Type,
		"created":     types.StringType,
		"published":   types.StringType,
		"touched":     types.StringType,
	}
	domainObjType := types.ObjectType{AttrTypes: domainObjAttrTypes}

	domainObjs := make([]attr.Value, 0, len(domains))
	for _, dom := range domains {
		obj, objDiags := types.ObjectValue(domainObjAttrTypes, map[string]attr.Value{
			"name":        types.StringValue(dom.Name),
			"minimum_ttl": types.Int64Value(int64(dom.MinimumTTL)),
			"created":     types.StringValue(dom.Created),
			"published":   types.StringValue(dom.Published),
			"touched":     types.StringValue(dom.Touched),
		})
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		domainObjs = append(domainObjs, obj)
	}

	domainsList, listDiags := types.ListValue(domainObjType, domainObjs)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Domains = domainsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
