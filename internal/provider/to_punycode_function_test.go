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

func TestToPunycodeFunction_Run(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		request  function.RunRequest
		expected function.RunResponse
	}{
		"unicode-umlaut": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("münchen.de")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("xn--mnchen-3ya.de")),
			},
		},
		"already-punycode": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("xn--mnchen-3ya.de")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("xn--mnchen-3ya.de")),
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
		"fqdn-trailing-dot": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("münchen.de.")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("xn--mnchen-3ya.de.")),
			},
		},
		"multi-label-unicode": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue("münchen.bücher.de")}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(types.StringValue("xn--mnchen-3ya.xn--bcher-kva.de")),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := function.RunResponse{
				Result: function.NewResultData(types.StringUnknown()),
			}
			(&toPunycodeFunction{}).Run(context.Background(), tc.request, &got)

			if diff := cmp.Diff(got, tc.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

// TestAccToPunycodeFunction_Valid verifies the function returns the correct
// Punycode result when called from a real Terraform configuration.
func TestAccToPunycodeFunction_Valid(t *testing.T) {
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
  value = provider::desec::to_punycode("münchen.de")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.StringExact("xn--mnchen-3ya.de")),
				},
			},
		},
	})
}

// TestAccToPunycodeFunction_ASCIIPassthrough verifies that a plain ASCII
// domain name is returned unchanged.
func TestAccToPunycodeFunction_ASCIIPassthrough(t *testing.T) {
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
  value = provider::desec::to_punycode("example.com")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.StringExact("example.com")),
				},
			},
		},
	})
}

// TestAccToPunycodeFunction_MultiLabel verifies that all labels in a
// multi-label unicode domain name are independently converted.
func TestAccToPunycodeFunction_MultiLabel(t *testing.T) {
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
  value = provider::desec::to_punycode("münchen.bücher.de")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.StringExact("xn--mnchen-3ya.xn--bcher-kva.de")),
				},
			},
		},
	})
}
