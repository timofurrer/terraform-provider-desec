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

// Ensure tokenListResource fully satisfies framework interfaces.
var _ list.ListResource = (*tokenListResource)(nil)
var _ list.ListResourceWithConfigure = (*tokenListResource)(nil)

// newTokenListResource creates a new tokenListResource.
func newTokenListResource() list.ListResource {
	return &tokenListResource{}
}

// tokenListResource lists deSEC API tokens.
type tokenListResource struct {
	client *api.Client
}

func (r *tokenListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *tokenListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists deSEC API authentication tokens for the current account.",
	}
}

func (r *tokenListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tokenListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	stream.Results = list.NoListResults

	tokens, err := r.client.ListTokens(ctx)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Error Listing Tokens",
				fmt.Sprintf("Unable to list tokens: %s", err)),
		})
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, token := range tokens {
			result := req.NewListResult(ctx)
			result.DisplayName = token.Name

			var model tokenResourceModel
			if diags := tokenToModel(ctx, &token, &model, false); diags.HasError() {
				result.Diagnostics.Append(diags...)
				push(result)
				return
			}

			if diags := result.Identity.Set(ctx, tokenIdentityModel{
				ID: types.StringValue(token.ID),
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
