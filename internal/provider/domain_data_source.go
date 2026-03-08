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

// Ensure DomainDataSource satisfies the datasource interface.
var _ datasource.DataSource = (*domainDataSource)(nil)

// newDomainDataSource creates a new DomainDataSource.
func newDomainDataSource() datasource.DataSource {
	return &domainDataSource{}
}

// domainDataSource reads a single deSEC domain by name.
type domainDataSource struct {
	client *api.Client
}

// domainDataSourceModel describes the data source data model.
type domainDataSourceModel struct {
	Name       types.String `tfsdk:"name"`
	ID         types.String `tfsdk:"id"`
	MinimumTTL types.Int64  `tfsdk:"minimum_ttl"`
	Created    types.String `tfsdk:"created"`
	Published  types.String `tfsdk:"published"`
	Touched    types.String `tfsdk:"touched"`
	Keys       types.List   `tfsdk:"keys"`
}

func (d *domainDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *domainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a specific deSEC DNS zone.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The domain name to look up.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The domain name, used as the data source identifier.",
				Computed:            true,
			},
			"minimum_ttl": schema.Int64Attribute{
				MarkdownDescription: "The minimum TTL (in seconds) that can be used for RRsets in this domain.",
				Computed:            true,
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Timestamp of domain creation in ISO 8601 format.",
				Computed:            true,
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

func (d *domainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *domainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := d.client.GetDomain(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Domain", fmt.Sprintf("Unable to read domain %q: %s", data.Name.ValueString(), err))
		return
	}

	// Reuse the shared converter (which fills the same fields).
	rm := &domainResourceModel{}
	diags := domainToModel(domain, rm)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = rm.ID
	data.Name = rm.Name
	data.MinimumTTL = rm.MinimumTTL
	data.Created = rm.Created
	data.Published = rm.Published
	data.Touched = rm.Touched
	data.Keys = rm.Keys

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
