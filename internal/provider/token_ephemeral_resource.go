// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	ephemerals "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure tokenEphemeralResource fully satisfies framework interfaces.
var _ ephemeral.EphemeralResource = (*tokenEphemeralResource)(nil)
var _ ephemeral.EphemeralResourceWithConfigure = (*tokenEphemeralResource)(nil)
var _ ephemeral.EphemeralResourceWithClose = (*tokenEphemeralResource)(nil)

// newTokenEphemeralResource creates a new tokenEphemeralResource.
func newTokenEphemeralResource() ephemeral.EphemeralResource {
	return &tokenEphemeralResource{}
}

// tokenEphemeralResource manages a short-lived deSEC API token.
type tokenEphemeralResource struct {
	client *api.Client
}

// tokenEphemeralResourceModel describes the ephemeral resource data model.
type tokenEphemeralResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Token            types.String `tfsdk:"token"`
	Created          types.String `tfsdk:"created"`
	LastUsed         types.String `tfsdk:"last_used"`
	Owner            types.String `tfsdk:"owner"`
	IsValid          types.Bool   `tfsdk:"is_valid"`
	PermCreateDomain types.Bool   `tfsdk:"perm_create_domain"`
	PermDeleteDomain types.Bool   `tfsdk:"perm_delete_domain"`
	PermManageTokens types.Bool   `tfsdk:"perm_manage_tokens"`
	AllowedSubnets   types.List   `tfsdk:"allowed_subnets"`
	AutoPolicy       types.Bool   `tfsdk:"auto_policy"`
	MaxAge           types.String `tfsdk:"max_age"`
	MaxUnusedPeriod  types.String `tfsdk:"max_unused_period"`
	KeepOnClose      types.Bool   `tfsdk:"keep_on_close"`
}

// tokenEphemeralPrivateState is stored in private state between Open and Close.
type tokenEphemeralPrivateState struct {
	TokenID     string `json:"token_id"`
	KeepOnClose bool   `json:"keep_on_close"`
}

func (r *tokenEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *tokenEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = ephemerals.Schema{
		MarkdownDescription: "Creates a short-lived deSEC API token on open and, by default, deletes it on close. Use `keep_on_close = true` to retain the token after Terraform finishes.",

		Attributes: map[string]ephemerals.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The token's UUID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable token name for reference purposes. Maximum 178 characters.",
				Optional:            true,
				Computed:            true,
			},
			"token": ephemerals.StringAttribute{
				MarkdownDescription: "The token's secret value used to authenticate API requests.",
				Computed:            true,
				Sensitive:           true,
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
				Optional:            true,
				Computed:            true,
			},
			"perm_delete_domain": schema.BoolAttribute{
				MarkdownDescription: "Whether this token may delete domains.",
				Optional:            true,
				Computed:            true,
			},
			"perm_manage_tokens": schema.BoolAttribute{
				MarkdownDescription: "Whether this token may manage tokens (list, create, modify, delete).",
				Optional:            true,
				Computed:            true,
			},
			"allowed_subnets": schema.ListAttribute{
				MarkdownDescription: "List of IP addresses or CIDR subnets that may authenticate using this token. Defaults to `[\"0.0.0.0/0\", \"::/0\"]` (unrestricted).",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"auto_policy": schema.BoolAttribute{
				MarkdownDescription: "When `true`, automatically creates a permissive scoping policy for each domain created with this token.",
				Optional:            true,
				Computed:            true,
			},
			"max_age": schema.StringAttribute{
				MarkdownDescription: "Maximum token lifetime as a duration string (e.g. `\"365 00:00:00\"`). Set to `null` for no age limit.",
				Optional:            true,
			},
			"max_unused_period": schema.StringAttribute{
				MarkdownDescription: "Maximum allowed period of disuse before the token is invalidated (e.g. `\"30 00:00:00\"`). Set to `null` for no limit.",
				Optional:            true,
			},
			"keep_on_close": schema.BoolAttribute{
				MarkdownDescription: "When `true`, the token is not deleted when Terraform closes the ephemeral resource. Defaults to `false`.",
				Optional:            true,
			},
		},
	}
}

func (r *tokenEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *tokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data tokenEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowedSubnets, diags := subnetsFromList(ctx, data.AllowedSubnets)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.CreateToken(
		ctx,
		data.Name.ValueString(),
		data.PermCreateDomain.ValueBool(),
		data.PermDeleteDomain.ValueBool(),
		data.PermManageTokens.ValueBool(),
		allowedSubnets,
		data.AutoPolicy.ValueBool(),
		nullableString(data.MaxAge),
		nullableString(data.MaxUnusedPeriod),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Ephemeral Token", fmt.Sprintf("Unable to create token: %s", err))
		return
	}

	data.ID = types.StringValue(token.ID)
	data.Name = types.StringValue(token.Name)
	data.Created = types.StringValue(token.Created)
	data.Owner = types.StringValue(token.Owner)
	data.IsValid = types.BoolValue(token.IsValid)
	data.PermCreateDomain = types.BoolValue(token.PermCreateDomain)
	data.PermDeleteDomain = types.BoolValue(token.PermDeleteDomain)
	data.PermManageTokens = types.BoolValue(token.PermManageTokens)
	data.AutoPolicy = types.BoolValue(token.AutoPolicy)

	if token.LastUsed != nil {
		data.LastUsed = types.StringValue(*token.LastUsed)
	} else {
		data.LastUsed = types.StringNull()
	}

	if token.MaxAge != nil {
		data.MaxAge = types.StringValue(*token.MaxAge)
	} else {
		data.MaxAge = types.StringNull()
	}

	if token.MaxUnusedPeriod != nil {
		data.MaxUnusedPeriod = types.StringValue(*token.MaxUnusedPeriod)
	} else {
		data.MaxUnusedPeriod = types.StringNull()
	}

	subnets, subnetDiags := types.ListValueFrom(ctx, types.StringType, token.AllowedSubnets)
	resp.Diagnostics.Append(subnetDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AllowedSubnets = subnets

	if token.Secret != "" {
		data.Token = types.StringValue(token.Secret)
	} else {
		data.Token = types.StringNull()
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Store token ID and keep_on_close in private state for Close.
	keepOnClose := !data.KeepOnClose.IsNull() && data.KeepOnClose.ValueBool()
	privateState := tokenEphemeralPrivateState{
		TokenID:     token.ID,
		KeepOnClose: keepOnClose,
	}
	privateBytes, err := json.Marshal(privateState)
	if err != nil {
		resp.Diagnostics.AddError("Error Encoding Private State", fmt.Sprintf("Unable to encode private state: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "state", privateBytes)...)
}

func (r *tokenEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	privateBytes, diags := req.Private.GetKey(ctx, "state")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(privateBytes) == 0 {
		return
	}

	var privateState tokenEphemeralPrivateState
	if err := json.Unmarshal(privateBytes, &privateState); err != nil {
		resp.Diagnostics.AddError("Error Decoding Private State", fmt.Sprintf("Unable to decode private state: %s", err))
		return
	}

	if privateState.KeepOnClose {
		return
	}

	if err := r.client.DeleteToken(ctx, privateState.TokenID); err != nil {
		if api.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Ephemeral Token", fmt.Sprintf("Unable to delete token %q: %s", privateState.TokenID, err))
	}
}
