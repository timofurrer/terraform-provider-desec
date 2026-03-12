// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure tokenPoliciesDataSource satisfies the datasource interface.
var _ datasource.DataSource = (*tokenPoliciesDataSource)(nil)

// newTokenPoliciesDataSource creates a new tokenPoliciesDataSource.
func newTokenPoliciesDataSource() datasource.DataSource {
	return &tokenPoliciesDataSource{}
}

// tokenPoliciesDataSource retrieves all scoping policies for a given token.
type tokenPoliciesDataSource struct {
	client *api.Client
}

// tokenPoliciesDataSourceModel describes the data source data model.
type tokenPoliciesDataSourceModel struct {
	TokenID  types.String `tfsdk:"token_id"`
	Policies types.List   `tfsdk:"policies"`
}

// tokenPolicyAttrTypes defines the attribute types for a policy object in the list.
var tokenPolicyAttrTypes = map[string]attr.Type{
	"id":         types.StringType,
	"domain":     types.StringType,
	"subname":    types.StringType,
	"type":       types.StringType,
	"perm_write": types.BoolType,
}

func (d *tokenPoliciesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token_policies"
}

func (d *tokenPoliciesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all scoping policies for a given deSEC API token.",

		Attributes: map[string]schema.Attribute{
			"token_id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the token whose policies to list.",
				Required:            true,
			},
			"policies": schema.ListNestedAttribute{
				MarkdownDescription: "List of all scoping policies for the token.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The policy's UUID.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *tokenPoliciesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tokenPoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tokenPoliciesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, err := d.client.ListTokenPolicies(ctx, data.TokenID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Token Policies",
			fmt.Sprintf("Unable to list token policies for token %q: %s", data.TokenID.ValueString(), err))
		return
	}

	policyObjType := types.ObjectType{AttrTypes: tokenPolicyAttrTypes}
	policyObjs := make([]attr.Value, 0, len(policies))

	for _, p := range policies {
		attrs := tokenPolicyToAttrValues(p)

		obj, objDiags := types.ObjectValue(tokenPolicyAttrTypes, attrs)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		policyObjs = append(policyObjs, obj)
	}

	policiesList, listDiags := types.ListValue(policyObjType, policyObjs)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Policies = policiesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// tokenPolicyToAttrValues converts an api.TokenPolicy to a map of attr.Value
// suitable for constructing a types.Object with tokenPolicyAttrTypes.
func tokenPolicyToAttrValues(p api.TokenPolicy) map[string]attr.Value {
	var domain, subname, rrtype attr.Value

	if p.Domain != nil {
		domain = types.StringValue(*p.Domain)
	} else {
		domain = types.StringNull()
	}

	if p.Subname != nil {
		subname = types.StringValue(*p.Subname)
	} else {
		subname = types.StringNull()
	}

	if p.Type != nil {
		rrtype = types.StringValue(*p.Type)
	} else {
		rrtype = types.StringNull()
	}

	return map[string]attr.Value{
		"id":         types.StringValue(p.ID),
		"domain":     domain,
		"subname":    subname,
		"type":       rrtype,
		"perm_write": types.BoolValue(p.PermWrite),
	}
}
