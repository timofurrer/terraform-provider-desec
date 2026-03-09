// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// Ensure tokenPolicyResource fully satisfies framework interfaces.
var _ resource.Resource = (*tokenPolicyResource)(nil)
var _ resource.ResourceWithImportState = (*tokenPolicyResource)(nil)
var _ resource.ResourceWithIdentity = (*tokenPolicyResource)(nil)

// tokenPolicyIdentityModel describes the identity of a token policy resource.
type tokenPolicyIdentityModel struct {
	TokenID types.String `tfsdk:"token_id"`
	ID      types.String `tfsdk:"id"`
}

func (r *tokenPolicyResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"token_id": identityschema.StringAttribute{
				Description:       "The UUID of the token this policy belongs to.",
				RequiredForImport: true,
			},
			"id": identityschema.StringAttribute{
				Description:       "The policy's UUID.",
				RequiredForImport: true,
			},
		},
	}
}

// newTokenPolicyResource creates a new tokenPolicyResource.
func newTokenPolicyResource() resource.Resource {
	return &tokenPolicyResource{}
}

// tokenPolicyResource manages a single deSEC token scoping policy.
type tokenPolicyResource struct {
	client *api.Client
}

// tokenPolicyResourceModel describes the resource data model.
type tokenPolicyResourceModel struct {
	ID        types.String `tfsdk:"id"`
	TokenID   types.String `tfsdk:"token_id"`
	Domain    types.String `tfsdk:"domain"`
	Subname   types.String `tfsdk:"subname"`
	Type      types.String `tfsdk:"type"`
	PermWrite types.Bool   `tfsdk:"perm_write"`
}

func (r *tokenPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token_policy"
}

func (r *tokenPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a scoping policy for a deSEC API token.\n\n" +
			"Token policies restrict which DNS records a token may write. A *default* policy " +
			"(all of `domain`, `subname`, and `type` unset / `null`) must exist before any " +
			"specific policies can be created, and it cannot be deleted while specific policies " +
			"are still in place. Use `depends_on` to enforce this ordering in your configuration.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The policy's UUID, used as the resource identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"token_id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the token this policy belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Domain name this policy applies to. Omit or set to `null` for the default (catch-all) policy.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subname": schema.StringAttribute{
				MarkdownDescription: "Subname (subdomain label) this policy applies to. Omit or set to `null` for a wildcard match on subname.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "DNS record type this policy applies to (e.g. `A`, `AAAA`, `TXT`). Omit or set to `null` for a wildcard match on type.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"perm_write": schema.BoolAttribute{
				MarkdownDescription: "Whether this policy grants write permission. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *tokenPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tokenPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data tokenPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.CreateTokenPolicy(ctx,
		data.TokenID.ValueString(),
		api.CreateTokenPolicyOptions{
			Domain:    nullableString(data.Domain),
			Subname:   nullableString(data.Subname),
			Type:      nullableString(data.Type),
			PermWrite: data.PermWrite.ValueBool(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Token Policy",
			fmt.Sprintf("Unable to create token policy for token %q: %s", data.TokenID.ValueString(), err))
		return
	}

	tokenPolicyToModel(policy, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, tokenPolicyIdentityModel{
		TokenID: data.TokenID,
		ID:      data.ID,
	})...)
}

func (r *tokenPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tokenPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetTokenPolicy(ctx, data.TokenID.ValueString(), data.ID.ValueString())
	if err != nil {
		if api.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Token Policy",
			fmt.Sprintf("Unable to read token policy %q for token %q: %s",
				data.ID.ValueString(), data.TokenID.ValueString(), err))
		return
	}

	tokenPolicyToModel(policy, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, tokenPolicyIdentityModel{
		TokenID: data.TokenID,
		ID:      data.ID,
	})...)
}

func (r *tokenPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data tokenPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state tokenPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.UpdateTokenPolicy(ctx,
		state.TokenID.ValueString(),
		state.ID.ValueString(),
		api.UpdateTokenPolicyOptions{
			PermWrite: data.PermWrite.ValueBool(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Token Policy",
			fmt.Sprintf("Unable to update token policy %q for token %q: %s",
				state.ID.ValueString(), state.TokenID.ValueString(), err))
		return
	}

	tokenPolicyToModel(policy, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tokenPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tokenPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTokenPolicy(ctx, data.TokenID.ValueString(), data.ID.ValueString()); err != nil {
		if api.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Token Policy",
			fmt.Sprintf("Unable to delete token policy %q for token %q: %s",
				data.ID.ValueString(), data.TokenID.ValueString(), err))
	}
}

func (r *tokenPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		// Import format: "token_id/policy_id"
		tokenID, policyID, ok := strings.Cut(req.ID, "/")
		if !ok {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Expected import ID in the format 'token_id/policy_id', got %q", req.ID),
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("token_id"), tokenID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), policyID)...)
		return
	}
	var identity tokenPolicyIdentityModel
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("token_id"), identity.TokenID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), identity.ID)...)
}

// tokenPolicyToModel converts an api.TokenPolicy into a tokenPolicyResourceModel.
func tokenPolicyToModel(p *api.TokenPolicy, m *tokenPolicyResourceModel) {
	m.ID = types.StringValue(p.ID)
	m.PermWrite = types.BoolValue(p.PermWrite)

	if p.Domain != nil {
		m.Domain = types.StringValue(*p.Domain)
	} else {
		m.Domain = types.StringNull()
	}

	if p.Subname != nil {
		m.Subname = types.StringValue(*p.Subname)
	} else {
		m.Subname = types.StringNull()
	}

	if p.Type != nil {
		m.Type = types.StringValue(*p.Type)
	} else {
		m.Type = types.StringNull()
	}
}
