// Copyright (c) Timo Furrer
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

// Ensure tokenPolicyListResource fully satisfies framework interfaces.
var _ list.ListResource = (*tokenPolicyListResource)(nil)
var _ list.ListResourceWithConfigure = (*tokenPolicyListResource)(nil)

// newTokenPolicyListResource creates a new tokenPolicyListResource.
func newTokenPolicyListResource() list.ListResource {
	return &tokenPolicyListResource{}
}

// tokenPolicyListResource lists deSEC token scoping policies.
type tokenPolicyListResource struct {
	client *api.Client
}

// tokenPolicyListConfigModel describes the required filter configuration for listing token policies.
type tokenPolicyListConfigModel struct {
	TokenID types.String `tfsdk:"token_id"`
}

func (r *tokenPolicyListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token_policy"
}

func (r *tokenPolicyListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists deSEC token scoping policies for a given API token.",
		Attributes: map[string]listschema.Attribute{
			"token_id": listschema.StringAttribute{
				MarkdownDescription: "The UUID of the token whose policies to list.",
				Required:            true,
			},
		},
	}
}

func (r *tokenPolicyListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tokenPolicyListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var config tokenPolicyListConfigModel
	stream.Results = list.NoListResults

	if diags := req.Config.Get(ctx, &config); diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	policies, err := r.client.ListTokenPolicies(ctx, config.TokenID.ValueString())
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Error Listing Token Policies",
				fmt.Sprintf("Unable to list token policies for token %q: %s", config.TokenID.ValueString(), err)),
		})
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, policy := range policies {
			result := req.NewListResult(ctx)
			result.DisplayName = policy.ID

			var model tokenPolicyResourceModel
			model.TokenID = config.TokenID
			tokenPolicyToModel(&policy, &model)

			if diags := result.Identity.Set(ctx, tokenPolicyIdentityModel{
				TokenID: config.TokenID,
				ID:      types.StringValue(policy.ID),
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
