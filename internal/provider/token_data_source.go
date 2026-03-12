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

// Ensure tokenDataSource satisfies the datasource interface.
var _ datasource.DataSource = (*tokenDataSource)(nil)

// newTokenDataSource creates a new tokenDataSource.
func newTokenDataSource() datasource.DataSource {
	return &tokenDataSource{}
}

// tokenDataSource reads a single deSEC token by ID.
type tokenDataSource struct {
	client *api.Client
}

// tokenDataSourceModel describes the data source data model.
// Note: the token secret is never returned by the GET endpoint, so it is not
// included here.
type tokenDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
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

func (d *tokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (d *tokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a specific deSEC API token by its ID. The token secret value is never returned by this data source.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The UUID of the token to look up.",
				Required:            true,
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
	}
}

func (d *tokenDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tokenDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := d.client.GetToken(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Token", fmt.Sprintf("Unable to read token %q: %s", data.ID.ValueString(), err))
		return
	}

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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
