// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
	"github.com/timofurrer/terraform-provider-desec/internal/api/fake"
)

// newIdentityTestClient starts a fresh in-process fake deSEC backend and
// returns an API client configured against it. Used by tests that call a
// resource's CRUD methods directly rather than going through the
// terraform-plugin-testing acceptance test harness.
func newIdentityTestClient(t *testing.T) *api.Client {
	t.Helper()
	srv := fake.NewServer()
	t.Cleanup(srv.Close)
	return api.NewClient(srv.URL(), srv.Token())
}

// configureTestResource wires up the given resource with the provided API
// client, exactly as the framework does before invoking any CRUD method.
func configureTestResource(t *testing.T, ctx context.Context, res resource.Resource, client *api.Client) {
	t.Helper()
	configurable, ok := res.(resource.ResourceWithConfigure)
	if !ok {
		t.Fatalf("%T does not implement resource.ResourceWithConfigure", res)
	}
	var resp resource.ConfigureResponse
	configurable.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("configure: %v", resp.Diagnostics)
	}
}

// resourceSchema fetches the resource's regular (non-identity) attribute schema.
func resourceSchema(ctx context.Context, res resource.Resource) resource.SchemaResponse {
	var resp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &resp)
	return resp
}

// nullResourceIdentity builds a resource identity value with every attribute
// null. This mirrors what the framework pre-populates a request/response
// identity with when there is no prior identity data available, e.g. state
// written before the resource supported identity.
func nullResourceIdentity(t *testing.T, ctx context.Context, res resource.Resource) *tfsdk.ResourceIdentity {
	t.Helper()
	identityRes, ok := res.(resource.ResourceWithIdentity)
	if !ok {
		t.Fatalf("%T does not implement resource.ResourceWithIdentity", res)
	}
	var idResp resource.IdentitySchemaResponse
	identityRes.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &idResp)
	nullVal := tftypes.NewValue(idResp.IdentitySchema.Type().TerraformType(ctx), nil)
	return &tfsdk.ResourceIdentity{Schema: idResp.IdentitySchema, Raw: nullVal}
}

// requireNonNullIdentity fails the test unless identity is present and
// populated, i.e. the resource itself called Identity.Set rather than
// relying on the framework already knowing the identity beforehand.
func requireNonNullIdentity(t *testing.T, identity *tfsdk.ResourceIdentity) {
	t.Helper()
	if identity == nil {
		t.Fatal("expected resource identity to be set, got nil")
	}
	if identity.Raw.IsFullyNull() {
		t.Fatal("expected resource identity to be populated, got a fully null value")
	}
}

func TestNullableString(t *testing.T) {
	t.Run("non-null value returns pointer to value", func(t *testing.T) {
		s := types.StringValue("hello")
		got := nullableString(s)
		if got == nil {
			t.Fatal("expected non-nil pointer, got nil")
		}
		if *got != "hello" {
			t.Fatalf("expected %q, got %q", "hello", *got)
		}
	})

	t.Run("empty string returns pointer to empty string", func(t *testing.T) {
		s := types.StringValue("")
		got := nullableString(s)
		if got == nil {
			t.Fatal("expected non-nil pointer, got nil")
		}
		if *got != "" {
			t.Fatalf("expected empty string, got %q", *got)
		}
	})

	t.Run("null returns nil", func(t *testing.T) {
		s := types.StringNull()
		got := nullableString(s)
		if got != nil {
			t.Fatalf("expected nil, got pointer to %q", *got)
		}
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		s := types.StringUnknown()
		got := nullableString(s)
		if got != nil {
			t.Fatalf("expected nil, got pointer to %q", *got)
		}
	})
}

func TestNullableBool(t *testing.T) {
	t.Run("true returns pointer to true", func(t *testing.T) {
		b := types.BoolValue(true)
		got := nullableBool(b)
		if got == nil {
			t.Fatal("expected non-nil pointer, got nil")
		}
		if *got != true {
			t.Fatalf("expected true, got %v", *got)
		}
	})

	t.Run("false returns pointer to false", func(t *testing.T) {
		b := types.BoolValue(false)
		got := nullableBool(b)
		if got == nil {
			t.Fatal("expected non-nil pointer, got nil")
		}
		if *got != false {
			t.Fatalf("expected false, got %v", *got)
		}
	})

	t.Run("null returns nil", func(t *testing.T) {
		b := types.BoolNull()
		got := nullableBool(b)
		if got != nil {
			t.Fatalf("expected nil, got pointer to %v", *got)
		}
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		b := types.BoolUnknown()
		got := nullableBool(b)
		if got != nil {
			t.Fatalf("expected nil, got pointer to %v", *got)
		}
	})
}
