// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure tokensDataSource satisfies the datasource interface.
var _ datasource.DataSource = (*tokensDataSource)(nil)

// newTokensDataSource creates a new tokensDataSource.
func newTokensDataSource() datasource.DataSource {
	return &tokensDataSource{}
}

// tokensDataSource retrieves all deSEC tokens for the account.
type tokensDataSource struct {
	client *api.Client
}

// tokensDataSourceModel describes the data source data model.
type tokensDataSourceModel struct {
	Tokens types.List `tfsdk:"tokens"`
}

// tokenAttrTypes defines the attribute types for a token object in the list.
var tokenAttrTypes = map[string]attr.Type{
	"id":                 types.StringType,
	"name":               types.StringType,
	"created":            types.StringType,
	"last_used":          types.StringType,
	"owner":              types.StringType,
	"is_valid":           types.BoolType,
	"perm_create_domain": types.BoolType,
	"perm_delete_domain": types.BoolType,
	"perm_manage_tokens": types.BoolType,
	"allowed_subnets":    types.ListType{ElemType: types.StringType},
	"auto_policy":        types.BoolType,
	"max_age":            types.StringType,
	"max_unused_period":  types.StringType,
}

func (d *tokensDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tokens"
}

func (d *tokensDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of all deSEC API tokens for the authenticated account. Token secret values are never included.",

		Attributes: map[string]schema.Attribute{
			"tokens": schema.ListNestedAttribute{
				MarkdownDescription: "List of all tokens in the account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The token's UUID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Human-readable token name.",
							Computed:            true,
						},
						"created": schema.StringAttribute{
							MarkdownDescription: "Timestamp of token creation in ISO 8601 format.",
							Computed:            true,
						},
						"last_used": schema.StringAttribute{
							MarkdownDescription: "Timestamp of when the token was last used, or `null` if it has never been used.",
							Computed:            true,
						},
						"owner": schema.StringAttribute{
							MarkdownDescription: "Email address of the deSEC account that owns this token.",
							Computed:            true,
						},
						"is_valid": schema.BoolAttribute{
							MarkdownDescription: "Whether the token is currently valid (not expired).",
							Computed:            true,
						},
						"perm_create_domain": schema.BoolAttribute{
							MarkdownDescription: "Whether this token may create new domains.",
							Computed:            true,
						},
						"perm_delete_domain": schema.BoolAttribute{
							MarkdownDescription: "Whether this token may delete domains.",
							Computed:            true,
						},
						"perm_manage_tokens": schema.BoolAttribute{
							MarkdownDescription: "Whether this token may manage tokens.",
							Computed:            true,
						},
						"allowed_subnets": schema.ListAttribute{
							MarkdownDescription: "List of IP addresses or CIDR subnets that may authenticate using this token.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"auto_policy": schema.BoolAttribute{
							MarkdownDescription: "Whether the token automatically creates a permissive scoping policy for each domain it creates.",
							Computed:            true,
						},
						"max_age": schema.StringAttribute{
							MarkdownDescription: "Maximum token lifetime as a duration string, or `null` if no age limit is set.",
							Computed:            true,
						},
						"max_unused_period": schema.StringAttribute{
							MarkdownDescription: "Maximum allowed period of disuse, or `null` if no limit is set.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *tokensDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tokensDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tokensDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tokens, err := d.client.ListTokens(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing Tokens", fmt.Sprintf("Unable to list tokens: %s", err))
		return
	}

	tokenObjType := types.ObjectType{AttrTypes: tokenAttrTypes}
	tokenObjs := make([]attr.Value, 0, len(tokens))

	for _, t := range tokens {
		attrs, attrDiags := tokenToAttrValues(ctx, t)
		resp.Diagnostics.Append(attrDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		obj, objDiags := types.ObjectValue(tokenAttrTypes, attrs)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		tokenObjs = append(tokenObjs, obj)
	}

	tokensList, listDiags := types.ListValue(tokenObjType, tokenObjs)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Tokens = tokensList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// tokenToAttrValues converts an api.Token to a map of attr.Value suitable for
// constructing a types.Object with tokenAttrTypes.
func tokenToAttrValues(ctx context.Context, t api.Token) (map[string]attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	var lastUsed attr.Value
	if t.LastUsed != nil {
		lastUsed = types.StringValue(*t.LastUsed)
	} else {
		lastUsed = types.StringNull()
	}

	var maxAge attr.Value
	if t.MaxAge != nil {
		maxAge = types.StringValue(*t.MaxAge)
	} else {
		maxAge = types.StringNull()
	}

	var maxUnusedPeriod attr.Value
	if t.MaxUnusedPeriod != nil {
		maxUnusedPeriod = types.StringValue(*t.MaxUnusedPeriod)
	} else {
		maxUnusedPeriod = types.StringNull()
	}

	subnetsList, subnetDiags := types.ListValueFrom(ctx, types.StringType, t.AllowedSubnets)
	diags.Append(subnetDiags...)
	if diags.HasError() {
		return nil, diags
	}

	return map[string]attr.Value{
		"id":                 types.StringValue(t.ID),
		"name":               types.StringValue(t.Name),
		"created":            types.StringValue(t.Created),
		"last_used":          lastUsed,
		"owner":              types.StringValue(t.Owner),
		"is_valid":           types.BoolValue(t.IsValid),
		"perm_create_domain": types.BoolValue(t.PermCreateDomain),
		"perm_delete_domain": types.BoolValue(t.PermDeleteDomain),
		"perm_manage_tokens": types.BoolValue(t.PermManageTokens),
		"allowed_subnets":    subnetsList,
		"auto_policy":        types.BoolValue(t.AutoPolicy),
		"max_age":            maxAge,
		"max_unused_period":  maxUnusedPeriod,
	}, diags
}
