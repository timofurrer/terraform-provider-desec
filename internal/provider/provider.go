// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure DesecProvider satisfies the provider interface.
var _ provider.Provider = (*desecProvider)(nil)
var _ provider.ProviderWithFunctions = (*desecProvider)(nil)
var _ provider.ProviderWithEphemeralResources = (*desecProvider)(nil)
var _ provider.ProviderWithListResources = (*desecProvider)(nil)

// desecProvider defines the provider implementation.
type desecProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// desecProviderModel describes the provider data model.
type desecProviderModel struct {
	APIToken          types.String `tfsdk:"api_token"`
	APIURL            types.String `tfsdk:"api_url"`
	MaxRetries        types.Int64  `tfsdk:"max_retries"`
	SerializeRequests types.Bool   `tfsdk:"serialize_requests"`
}

func (p *desecProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "desec"
	resp.Version = p.version
}

func (p *desecProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The [deSEC](https://desec.io) provider enables Terraform and OpenTofu to manage DNS infrastructure hosted on deSEC, a free and secure DNS hosting service.

Use this provider to declaratively manage:

- **Domains** — create and inspect DNS zones, including DNSSEC key material ([deSEC domains API](https://desec.readthedocs.io/en/latest/dns/domains.html))
- **Resource Record Sets** — manage any DNS record type at the zone apex or any subdomain ([deSEC RRsets API](https://desec.readthedocs.io/en/latest/dns/rrsets.html))
- **API Tokens** — create scoped, expiring authentication tokens with fine-grained IP allowlists and domain permissions ([deSEC tokens API](https://desec.readthedocs.io/en/latest/auth/tokens.html))
- **Token Policies** — define per-domain, per-subname, and per-type write permissions for tokens, including default catch-all policies

### Guides

- [Getting Started](guides/01-getting-started) — register a domain, access nameservers and DNSSEC material, and create DNS records.
- [Migrating to TF with Bulk Import and Config Bootstrapping](guides/02-migrating-with-bulk-import) — discover existing resources and generate configuration to bring them under TF management.
- [Auditing with List Resources](guides/03-auditing-with-list-resources) — enumerate domains, records, tokens, and policies in your deSEC account.

### Data Sources

Read-only access is available for all of the above through matching data sources (` + "`" + `desec_domain` + "`" + `, ` + "`" + `desec_domains` + "`" + `, ` + "`" + `desec_record` + "`" + `, ` + "`" + `desec_records` + "`" + `, ` + "`" + `desec_token` + "`" + `, ` + "`" + `desec_tokens` + "`" + `, ` + "`" + `desec_token_policy` + "`" + `, ` + "`" + `desec_token_policies` + "`" + `), plus a ` + "`" + `desec_zonefile` + "`" + ` data source that exports a full RFC 1035 zone file.

### Ephemeral Resources

` + "`" + `ephemeral "desec_token"` + "`" + ` creates a short-lived token for use within a single Terraform/OpenTofu run. By default the token is deleted on close; set ` + "`" + `keep_on_close = true` + "`" + ` to retain it after the run completes.

### List Resources

` + "`" + `desec_domain` + "`" + `, ` + "`" + `desec_record` + "`" + `, ` + "`" + `desec_token` + "`" + `, and ` + "`" + `desec_token_policy` + "`" + ` are also available as list resources for bulk import and enumeration workflows.

### Functions

` + "`" + `provider::desec::to_punycode()` + "`" + ` and ` + "`" + `provider::desec::from_punycode()` + "`" + ` convert domain names between unicode (human-readable) and [Punycode](https://en.wikipedia.org/wiki/Punycode) (ACE) form. The deSEC API only accepts domain names in Punycode form, so these functions are useful when working with Internationalized Domain Names (IDN) containing non-ASCII characters such as umlauts.

### Rate Limiting

The deSEC API enforces rate limits. The provider automatically retries requests that receive HTTP 429 responses, honouring the ` + "`" + `Retry-After` + "`" + ` header (up to 5 retries per request by default, configurable via ` + "`" + `max_retries` + "`" + `). See the [deSEC rate limits documentation](https://desec.readthedocs.io/en/latest/rate-limits.html).

To prevent bursts of concurrent Terraform operations from exhausting deSEC rate limit buckets, the provider serializes API requests by default: domain-scoped requests (RRset operations, per-domain zone operations) are serialized per domain, and global DNS operations (domain creation/listing) share a single lock. This ensures at most one in-flight request per domain at a time, fully respecting the ` + "`" + `dns_api_per_domain_expensive` + "`" + ` (2/s per domain) and ` + "`" + `dns_api_expensive` + "`" + ` (10/s global) limits regardless of Terraform's parallelism setting. Serialization can be disabled via ` + "`" + `serialize_requests = false` + "`" + ` if you manage concurrency externally (e.g. via ` + "`" + `-parallelism=1` + "`" + `).`,
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "The deSEC API token. Can also be set via the `DESEC_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The deSEC API base URL. Defaults to `https://desec.io/api/v1`. Can also be set via the `DESEC_API_URL` environment variable. Can be overridden for custom endpoints or testing.",
				Optional:            true,
			},
			"max_retries": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of times a request that receives an HTTP 429 response is retried. Defaults to `5`.",
				Optional:            true,
			},
			"serialize_requests": schema.BoolAttribute{
				MarkdownDescription: "Serialize concurrent API requests to avoid hitting deSEC rate limits. When `true` (the default), domain-scoped requests are serialized per domain and global DNS requests share a single lock. Set to `false` only if you manage concurrency externally (e.g. via `-parallelism=1`).",
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
	// If the token is unknown at plan time (e.g. sourced from another resource
	// that hasn't been created yet), defer configuration to apply time by
	// returning early without error. Resources guard against a nil client via
	// their own Configure methods.
	if data.APIToken.IsUnknown() {
		return
	}
	apiToken := os.Getenv("DESEC_API_TOKEN")
	if !data.APIToken.IsNull() {
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

	var clientOpts []api.ClientOption
	if !data.MaxRetries.IsNull() && !data.MaxRetries.IsUnknown() {
		clientOpts = append(clientOpts, api.WithMaxRetries(int(data.MaxRetries.ValueInt64())))
	}
	if !data.SerializeRequests.IsNull() && !data.SerializeRequests.IsUnknown() {
		clientOpts = append(clientOpts, api.WithSerializeRequests(data.SerializeRequests.ValueBool()))
	}

	client := api.NewClient(apiURL, apiToken, clientOpts...)
	resp.DataSourceData = client
	resp.ResourceData = client
	resp.EphemeralResourceData = client
	resp.ListResourceData = client
}

func (p *desecProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		newToPunycodeFunction,
		newFromPunycodeFunction,
	}
}

func (p *desecProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newDomainResource,
		newRecordResource,
		newRecordsResource,
		newTokenResource,
		newTokenPolicyResource,
	}
}

func (p *desecProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		newTokenEphemeralResource,
	}
}

func (p *desecProvider) ListResources(_ context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		newDomainListResource,
		newRecordListResource,
		newTokenListResource,
		newTokenPolicyListResource,
	}
}

func (p *desecProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newDomainDataSource,
		newDomainsDataSource,
		newRecordDataSource,
		newRecordsDataSource,
		newZonefileDataSource,
		newTokenDataSource,
		newTokensDataSource,
		newTokenPolicyDataSource,
		newTokenPoliciesDataSource,
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
