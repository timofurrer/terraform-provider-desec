// Copyright (c) Timo Furrer
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

// Ensure ZonefileDataSource satisfies the datasource interface.
var _ datasource.DataSource = (*zonefileDataSource)(nil)

// newZonefileDataSource creates a new ZonefileDataSource.
func newZonefileDataSource() datasource.DataSource {
	return &zonefileDataSource{}
}

// zonefileDataSource exports a domain as a zonefile.
type zonefileDataSource struct {
	client *api.Client
}

// zonefileDataSourceModel describes the data source data model.
type zonefileDataSourceModel struct {
	Name     types.String `tfsdk:"name"`
	Zonefile types.String `tfsdk:"zonefile"`
}

func (d *zonefileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zonefile"
}

func (d *zonefileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Exports a deSEC DNS zone in RFC 1035 / BIND zone file format.\n\n" +
			"The returned content is plain text. Each resource record is on its own line in the form:\n\n" +
			"```\n<name>  <ttl>  IN  <type>  <rdata>\n```\n\n" +
			"The export includes a comment header line with the domain name and export timestamp. " +
			"DNSSEC-specific record types (RRSIG, NSEC, NSEC3, etc.) are excluded — only user-managed record types are present.\n\n" +
			"See the [deSEC API documentation](https://desec.readthedocs.io/en/latest/dns/domains.html#exporting-a-domain-as-zonefile) for further details.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The domain name to export.",
				Required:            true,
			},
			"zonefile": schema.StringAttribute{
				MarkdownDescription: "The zone file content in RFC 1035 / BIND format. Includes all user-managed RRsets. DNSSEC-related types are excluded.",
				Computed:            true,
			},
		},
	}
}

func (d *zonefileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *zonefileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data zonefileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zonefile, err := d.client.GetZonefile(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Zonefile", fmt.Sprintf("Unable to get zonefile for %q: %s", data.Name.ValueString(), err))
		return
	}

	data.Zonefile = types.StringValue(zonefile)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
