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

// Ensure recordListResource fully satisfies framework interfaces.
var _ list.ListResource = (*recordListResource)(nil)
var _ list.ListResourceWithConfigure = (*recordListResource)(nil)

// newRecordListResource creates a new recordListResource.
func newRecordListResource() list.ListResource {
	return &recordListResource{}
}

// recordListResource lists deSEC RRsets within a domain.
type recordListResource struct {
	client *api.Client
}

// recordListConfigModel describes the filter configuration for listing records.
type recordListConfigModel struct {
	Domain  types.String `tfsdk:"domain"`
	Subname types.String `tfsdk:"subname"`
	Type    types.String `tfsdk:"type"`
}

func (r *recordListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_record"
}

func (r *recordListResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists deSEC DNS resource record sets (RRsets) within a domain.",
		Attributes: map[string]listschema.Attribute{
			"domain": listschema.StringAttribute{
				MarkdownDescription: "The domain name to list records for.",
				Required:            true,
			},
			"subname": listschema.StringAttribute{
				MarkdownDescription: "Filter by subdomain part of the record name.",
				Optional:            true,
			},
			"type": listschema.StringAttribute{
				MarkdownDescription: "Filter by DNS record type (e.g. `A`, `AAAA`, `TXT`).",
				Optional:            true,
			},
		},
	}
}

func (r *recordListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *recordListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var config recordListConfigModel
	stream.Results = list.NoListResults

	if diags := req.Config.Get(ctx, &config); diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	opts := api.ListRRsetsOptions{
		Subname: nullableString(config.Subname),
		Type:    nullableString(config.Type),
	}

	rrsets, err := r.client.ListRRsets(ctx, config.Domain.ValueString(), opts)
	if err != nil {
		stream.Results = list.ListResultsStreamDiagnostics(diag.Diagnostics{
			diag.NewErrorDiagnostic("Error Listing Records",
				fmt.Sprintf("Unable to list records for domain %q: %s", config.Domain.ValueString(), err)),
		})
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, rrset := range rrsets {
			result := req.NewListResult(ctx)

			var model recordResourceModel
			if diags := rrsetToModel(ctx, &rrset, &model); diags.HasError() {
				result.Diagnostics.Append(diags...)
				push(result)
				return
			}

			result.DisplayName = model.Domain.ValueString() + "/" + model.Subname.ValueString() + "/" + model.Type.ValueString()

			if diags := result.Identity.Set(ctx, recordIdentityModel{
				Domain:  model.Domain,
				Subname: types.StringValue(model.Subname.ValueString()),
				Type:    model.Type,
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
