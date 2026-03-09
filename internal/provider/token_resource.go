// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure tokenResource fully satisfies framework interfaces.
var _ resource.Resource = (*tokenResource)(nil)
var _ resource.ResourceWithImportState = (*tokenResource)(nil)
var _ resource.ResourceWithIdentity = (*tokenResource)(nil)

// tokenIdentityModel describes the identity of a token resource.
type tokenIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func (r *tokenResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				Description:       "The token's UUID.",
				RequiredForImport: true,
			},
		},
	}
}

// newTokenResource creates a new tokenResource.
func newTokenResource() resource.Resource {
	return &tokenResource{}
}

// tokenResource manages a deSEC API token.
type tokenResource struct {
	client *api.Client
}

// tokenResourceModel describes the resource data model.
type tokenResourceModel struct {
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
}

func (r *tokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *tokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a deSEC API authentication token. The token secret value is only available at creation time and is stored in Terraform state as a sensitive value. After import, the token secret is unavailable.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The token's UUID, used as the resource identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable token name for reference purposes. Maximum 178 characters.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The token's secret value used to authenticate API requests. Only available immediately after creation; cannot be recovered afterwards. Set to `null` after import.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Timestamp of token creation in ISO 8601 format.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_used": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the token was last used, or `null` if it has never been used.",
				Computed:            true,
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "Email address of the deSEC account that owns this token.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_valid": schema.BoolAttribute{
				MarkdownDescription: "Whether the token is currently valid (not expired).",
				Computed:            true,
			},
			"perm_create_domain": schema.BoolAttribute{
				MarkdownDescription: "Whether this token may create new domains.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"perm_delete_domain": schema.BoolAttribute{
				MarkdownDescription: "Whether this token may delete domains.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"perm_manage_tokens": schema.BoolAttribute{
				MarkdownDescription: "Whether this token may manage tokens (list, create, modify, delete).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allowed_subnets": schema.ListAttribute{
				MarkdownDescription: "List of IP addresses or CIDR subnets that may authenticate using this token. Defaults to `[\"0.0.0.0/0\", \"::/0\"]` (unrestricted).",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_policy": schema.BoolAttribute{
				MarkdownDescription: "When `true`, automatically creates a permissive scoping policy for each domain created with this token.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"max_age": schema.StringAttribute{
				MarkdownDescription: "Maximum token lifetime as a duration string (e.g. `\"365 00:00:00\"`). Set to `null` for no age limit.",
				Optional:            true,
			},
			"max_unused_period": schema.StringAttribute{
				MarkdownDescription: "Maximum allowed period of disuse before the token is invalidated (e.g. `\"30 00:00:00\"`). Set to `null` for no limit.",
				Optional:            true,
			},
		},
	}
}

func (r *tokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *tokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data tokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowedSubnets, diags := subnetsFromList(ctx, data.AllowedSubnets)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.CreateToken(ctx, api.CreateTokenOptions{
		Name:             nullableString(data.Name),
		PermCreateDomain: nullableBool(data.PermCreateDomain),
		PermDeleteDomain: nullableBool(data.PermDeleteDomain),
		PermManageTokens: nullableBool(data.PermManageTokens),
		AllowedSubnets:   allowedSubnets,
		AutoPolicy:       nullableBool(data.AutoPolicy),
		MaxAge:           nullableString(data.MaxAge),
		MaxUnusedPeriod:  nullableString(data.MaxUnusedPeriod),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Token", fmt.Sprintf("Unable to create token: %s", err))
		return
	}

	resp.Diagnostics.Append(tokenToModel(ctx, token, &data, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, tokenIdentityModel{
		ID: types.StringValue(token.ID),
	})...)
}

func (r *tokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.GetToken(ctx, data.ID.ValueString())
	if err != nil {
		if api.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Token", fmt.Sprintf("Unable to read token %q: %s", data.ID.ValueString(), err))
		return
	}

	// Preserve the existing token secret — the API never returns it after creation.
	resp.Diagnostics.Append(tokenToModel(ctx, token, &data, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, tokenIdentityModel{
		ID: types.StringValue(token.ID),
	})...)
}

func (r *tokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data tokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the current state to get the ID and preserve the token secret.
	var state tokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowedSubnets, diags := subnetsFromList(ctx, data.AllowedSubnets)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.UpdateToken(ctx, state.ID.ValueString(), api.UpdateTokenOptions{
		Name:             nullableString(data.Name),
		PermCreateDomain: nullableBool(data.PermCreateDomain),
		PermDeleteDomain: nullableBool(data.PermDeleteDomain),
		PermManageTokens: nullableBool(data.PermManageTokens),
		AllowedSubnets:   allowedSubnets,
		AutoPolicy:       nullableBool(data.AutoPolicy),
		MaxAge:           nullableString(data.MaxAge),
		MaxUnusedPeriod:  nullableString(data.MaxUnusedPeriod),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Token", fmt.Sprintf("Unable to update token %q: %s", state.ID.ValueString(), err))
		return
	}

	// Preserve the token secret from state; Update response never includes it.
	resp.Diagnostics.Append(tokenToModel(ctx, token, &data, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Token = state.Token

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteToken(ctx, data.ID.ValueString()); err != nil {
		if api.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Token", fmt.Sprintf("Unable to delete token %q: %s", data.ID.ValueString(), err))
	}
}

func (r *tokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

// tokenToModel converts an api.Token into a tokenResourceModel.
// When preserveSecret is true, the Token field is left unchanged (keeping the
// value already in the model) since the API does not return the secret on reads.
func tokenToModel(ctx context.Context, t *api.Token, m *tokenResourceModel, preserveSecret bool) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(t.ID)
	m.Name = types.StringValue(t.Name)
	m.Created = types.StringValue(t.Created)
	m.Owner = types.StringValue(t.Owner)
	m.IsValid = types.BoolValue(t.IsValid)
	m.PermCreateDomain = types.BoolValue(t.PermCreateDomain)
	m.PermDeleteDomain = types.BoolValue(t.PermDeleteDomain)
	m.PermManageTokens = types.BoolValue(t.PermManageTokens)
	m.AutoPolicy = types.BoolValue(t.AutoPolicy)

	if t.LastUsed != nil {
		m.LastUsed = types.StringValue(*t.LastUsed)
	} else {
		m.LastUsed = types.StringNull()
	}

	if t.MaxAge != nil {
		m.MaxAge = types.StringValue(*t.MaxAge)
	} else {
		m.MaxAge = types.StringNull()
	}

	if t.MaxUnusedPeriod != nil {
		m.MaxUnusedPeriod = types.StringValue(*t.MaxUnusedPeriod)
	} else {
		m.MaxUnusedPeriod = types.StringNull()
	}

	subnets, subnetDiags := types.ListValueFrom(ctx, types.StringType, t.AllowedSubnets)
	diags.Append(subnetDiags...)
	if diags.HasError() {
		return diags
	}
	m.AllowedSubnets = subnets

	if !preserveSecret {
		if t.Secret != "" {
			m.Token = types.StringValue(t.Secret)
		} else {
			m.Token = types.StringNull()
		}
	}

	return diags
}

// subnetsFromList converts a types.List into a []string of subnet strings.
// If the list is null or unknown, returns nil (let the API use its default).
func subnetsFromList(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	var subnets []string
	diags := l.ElementsAs(ctx, &subnets, false)
	return subnets, diags
}
