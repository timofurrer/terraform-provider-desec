// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestFromPunycodeFunction_Run(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		request  function.RunRequest
		expected function.RunResponse
	}{
		"punycode-to-unicode": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("xn--mnchen-3ya.de")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("münchen.de")),
			},
		},
		"ascii-passthrough": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("example.com")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("example.com")),
			},
		},
		"already-unicode": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("münchen.de")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("münchen.de")),
			},
		},
		"fqdn-trailing-dot": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("xn--mnchen-3ya.de.")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("münchen.de.")),
			},
		},
		"multi-label": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("xn--mnchen-3ya.xn--bcher-kva.de")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("münchen.bücher.de")),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := function.RunResponse{
				Result: function.NewResultData(types.StringUnknown()),
			}
			(&fromPunycodeFunction{}).Run(context.Background(), tc.request, &got)

			if diff := cmp.Diff(got, tc.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

// TestAccFromPunycodeFunction_Valid verifies the function decodes a Punycode
// domain name back to its unicode form when called from a real Terraform
// configuration.
func TestAccFromPunycodeFunction_Valid(t *testing.T) {
	_, factories := newTestAccEnv(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
output "result" {
  value = provider::desec::from_punycode("xn--mnchen-3ya.de")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.StringExact("münchen.de")),
				},
			},
		},
	})
}

// TestAccFromPunycodeFunction_ASCIIPassthrough verifies that a plain ASCII
// domain name is returned unchanged.
func TestAccFromPunycodeFunction_ASCIIPassthrough(t *testing.T) {
	_, factories := newTestAccEnv(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
output "result" {
  value = provider::desec::from_punycode("example.com")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.StringExact("example.com")),
				},
			},
		},
	})
}

// TestAccFromPunycodeFunction_MultiLabel verifies that all xn-- labels in a
// multi-label domain name are independently decoded.
func TestAccFromPunycodeFunction_MultiLabel(t *testing.T) {
	_, factories := newTestAccEnv(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
output "result" {
  value = provider::desec::from_punycode("xn--mnchen-3ya.xn--bcher-kva.de")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.StringExact("münchen.bücher.de")),
				},
			},
		},
	})
}

// TestAccFromPunycodeFunction_RoundTrip verifies that encoding then decoding a
// unicode domain name returns the original value.
func TestAccFromPunycodeFunction_RoundTrip(t *testing.T) {
	_, factories := newTestAccEnv(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
output "result" {
  value = provider::desec::from_punycode(provider::desec::to_punycode("münchen.de"))
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.StringExact("münchen.de")),
				},
			},
		},
	})
}
