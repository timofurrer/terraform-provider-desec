// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

var _ datasource.DataSource = (*rrsetDataSource)(nil)

func newRRsetDataSource() datasource.DataSource {
	return &rrsetDataSource{}
}

type rrsetDataSource struct {
	client *api.Client
}

type rrsetDataSourceModel struct {
	Domain  types.String `tfsdk:"domain"`
	Subname types.String `tfsdk:"subname"`
	Type    types.String `tfsdk:"type"`
	TTL     types.Int64  `tfsdk:"ttl"`
	RData   types.Set    `tfsdk:"rdata"`
	Name    types.String `tfsdk:"name"`
	Created types.String `tfsdk:"created"`
	Touched types.String `tfsdk:"touched"`
}

func (d *rrsetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rrset"
}

func (d *rrsetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a specific DNS Resource Record Set (RRset) from a deSEC domain.",

		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain name.",
				Required:            true,
			},
			"subname": schema.StringAttribute{
				MarkdownDescription: "The subdomain part of the RRset name. Use `@` for the zone apex.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The DNS record type (e.g. `A`, `AAAA`, `TXT`).",
				Required:            true,
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "The TTL (time-to-live) in seconds.",
				Computed:            true,
			},
			"rdata": schema.SetAttribute{
				MarkdownDescription: "The set of RDATA strings.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The full DNS name of the RRset.",
				Computed:            true,
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Timestamp of RRset creation in ISO 8601 format.",
				Computed:            true,
			},
			"touched": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the RRset was last touched.",
				Computed:            true,
			},
		},
	}
}

func (d *rrsetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *rrsetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data rrsetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rrset, err := d.client.GetRRset(ctx,
		data.Domain.ValueString(),
		data.Subname.ValueString(),
		data.Type.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading RRset",
			fmt.Sprintf("Unable to read RRset %s/%s/%s: %s",
				data.Domain.ValueString(), data.Subname.ValueString(), data.Type.ValueString(), err))
		return
	}

	data.Domain = types.StringValue(rrset.Domain)
	data.Subname = types.StringValue(normalizeSubname(rrset.Subname))
	data.Type = types.StringValue(rrset.Type)
	data.TTL = types.Int64Value(int64(rrset.TTL))
	data.Name = types.StringValue(rrset.Name)
	data.Created = types.StringValue(rrset.Created)
	data.Touched = types.StringValue(rrset.Touched)

	setVal, diags := types.SetValueFrom(ctx, types.StringType, rrset.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RData = setVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
