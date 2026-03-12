// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure domainListResource fully satisfies framework interfaces.
var _ list.ListResource = (*domainListResource)(nil)
var _ list.ListResourceWithConfigure = (*domainListResource)(nil)

// newDomainListResource creates a new domainListResource.
func newDomainListResource() list.ListResource {
	return &domainListResource{}
}

// domainListResource lists deSEC DNS zones.
type domainListResource struct {
	client *api.Client
}

// domainListConfigModel describes the optional filter configuration for listing domains.
type domainListConfigModel struct {
	OwnsQname types.String `tfsdk:"owns_qname"`
}

func (r *domainListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists deSEC DNS zones (domains).",
		Attributes: map[string]listschema.Attribute{
			"owns_qname": listschema.StringAttribute{
				MarkdownDescription: "Filter domains by whether they own the given fully-qualified domain name.",
				Optional:            true,
			},
		},
	}
}

func (r *domainListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected List Resource Configure Type",
			fmt.Sprintf("Expected *api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *domainListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var config domainListConfigModel
	stream.Results = list.NoListResults

	if diags := req.Config.Get(ctx, &config); diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	opts := api.ListDomainsOptions{
		OwnsQname: nullableString(config.OwnsQname),
	}

	domains, err := r.client.ListDomains(ctx, opts)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Error Listing Domains",
				fmt.Sprintf("Unable to list domains: %s", err)),
		})
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, domain := range domains {
			result := req.NewListResult(ctx)
			result.DisplayName = domain.Name

			var model domainResourceModel
			if diags := domainToModel(&domain, &model); diags.HasError() {
				result.Diagnostics.Append(diags...)
				push(result)
				return
			}

			if diags := result.Identity.Set(ctx, domainIdentityModel{
				Name: types.StringValue(domain.Name),
			}); diags.HasError() {
				result.Diagnostics.Append(diags...)
				push(result)
				return
			}

			if req.IncludeResource {
				if diags := result.Resource.Set(ctx, model); diags.HasError() {
					result.Diagnostics.Append(diags...)
					push(result)
					return
				}
			}

			if !push(result) {
				return
			}
		}
	}
}
