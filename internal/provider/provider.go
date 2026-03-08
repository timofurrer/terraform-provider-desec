// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure DesecProvider satisfies the provider interface.
var _ provider.Provider = (*desecProvider)(nil)

// desecProvider defines the provider implementation.
type desecProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// desecProviderModel describes the provider data model.
type desecProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
	APIURL   types.String `tfsdk:"api_url"`
}

func (p *desecProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "desec"
	resp.Version = p.version
}

func (p *desecProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The deSEC provider manages DNS zones and resource record sets using the [deSEC DNS API](https://desec.io).",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "The deSEC API token. Can also be set via the `DESEC_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The deSEC API base URL. Defaults to `https://desec.io/api/v1`. Can also be set via the `DESEC_API_URL` environment variable. Override this for testing.",
				Optional:            true,
			},
		},
	}
}

func (p *desecProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data desecProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve API token from config or environment.
	apiToken := os.Getenv("DESEC_API_TOKEN")
	if !data.APIToken.IsNull() && !data.APIToken.IsUnknown() {
		apiToken = data.APIToken.ValueString()
	}
	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"The deSEC provider requires an API token. Set the `api_token` attribute or the `DESEC_API_TOKEN` environment variable.",
		)
		return
	}

	// Resolve API URL from config or environment.
	apiURL := os.Getenv("DESEC_API_URL")
	if apiURL == "" {
		apiURL = api.DefaultBaseURL
	}
	if !data.APIURL.IsNull() && !data.APIURL.IsUnknown() {
		apiURL = data.APIURL.ValueString()
	}

	client := api.NewClient(apiURL, apiToken)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *desecProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newDomainResource,
		newRecordResource,
	}
}

func (p *desecProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newDomainDataSource,
		newDomainsDataSource,
		newRecordDataSource,
		newRecordsDataSource,
		newZonefileDataSource,
	}
}

// New returns a function that creates the deSEC provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &desecProvider{
			version: version,
		}
	}
}
