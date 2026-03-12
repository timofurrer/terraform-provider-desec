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

// Ensure tokenPolicyDataSource satisfies the datasource interface.
var _ datasource.DataSource = (*tokenPolicyDataSource)(nil)

// newTokenPolicyDataSource creates a new tokenPolicyDataSource.
func newTokenPolicyDataSource() datasource.DataSource {
	return &tokenPolicyDataSource{}
}

// tokenPolicyDataSource reads a single deSEC token policy by token ID + policy ID.
type tokenPolicyDataSource struct {
	client *api.Client
}

// tokenPolicyDataSourceModel describes the data source data model.
type tokenPolicyDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	TokenID   types.String `tfsdk:"token_id"`
	Domain    types.String `tfsdk:"domain"`
	Subname   types.String `tfsdk:"subname"`
	Type      types.String `tfsdk:"type"`
	PermWrite types.Bool   `tfsdk:"perm_write"`
}

func (d *tokenPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token_policy"
}

func (d *tokenPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a specific deSEC token scoping policy.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the policy to look up.",
				Required:            true,
			},
			"token_id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the token this policy belongs to.",
				Required:            true,
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Domain name this policy applies to, or `null` for the default policy.",
				Computed:            true,
			},
			"subname": schema.StringAttribute{
				MarkdownDescription: "Subname this policy applies to, or `null` for a wildcard match.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "DNS record type this policy applies to, or `null` for a wildcard match.",
				Computed:            true,
			},
			"perm_write": schema.BoolAttribute{
				MarkdownDescription: "Whether this policy grants write permission.",
				Computed:            true,
			},
		},
	}
}

func (d *tokenPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tokenPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tokenPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := d.client.GetTokenPolicy(ctx, data.TokenID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Token Policy",
			fmt.Sprintf("Unable to read token policy %q for token %q: %s",
				data.ID.ValueString(), data.TokenID.ValueString(), err))
		return
	}

	data.PermWrite = types.BoolValue(policy.PermWrite)

	if policy.Domain != nil {
		data.Domain = types.StringValue(*policy.Domain)
	} else {
		data.Domain = types.StringNull()
	}

	if policy.Subname != nil {
		data.Subname = types.StringValue(*policy.Subname)
	} else {
		data.Subname = types.StringNull()
	}

	if policy.Type != nil {
		data.Type = types.StringValue(*policy.Type)
	} else {
		data.Type = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
