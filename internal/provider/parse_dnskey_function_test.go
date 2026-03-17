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

func mustDNSKeyObject(flags, protocol, algorithm int64, publicKey string) types.Object {
	v, diags := types.ObjectValue(dnskeyAttrTypes, map[string]attr.Value{
		"flags":      types.Int64Value(flags),
		"protocol":   types.Int64Value(protocol),
		"algorithm":  types.Int64Value(algorithm),
		"public_key": types.StringValue(publicKey),
	})
	if diags.HasError() {
		panic("failed to construct DNSKEY object: " + diags.Errors()[0].Detail())
	}
	return v
}

func TestParseDNSKeyFunction_Run(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		request  function.RunRequest
		expected function.RunResponse
	}{
		"ksk-ecdsap256sha256": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("257 3 13 MemvLhocKfZ8t/7kAef1UJ3cZGjkZLM3c+e76HZ2d2z7EW+6J2EbhHWAUcHhH/JZ5PtNi0GpEy5U56WvLswZAA=="),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustDNSKeyObject(257, 3, 13, "MemvLhocKfZ8t/7kAef1UJ3cZGjkZLM3c+e76HZ2d2z7EW+6J2EbhHWAUcHhH/JZ5PtNi0GpEy5U56WvLswZAA==")),
			},
		},
		"zsk-rsa": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("256 3 8 AwEAAd...base64...=="),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustDNSKeyObject(256, 3, 8, "AwEAAd...base64...==")),
			},
		},
		"public-key-with-spaces": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("257 3 13 abc def ghi"),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustDNSKeyObject(257, 3, 13, "abc def ghi")),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := function.RunResponse{
				Result: function.NewResultData(types.ObjectUnknown(dnskeyAttrTypes)),
			}
			(&parseDNSKeyFunction{}).Run(context.Background(), tc.request, &got)

			if diff := cmp.Diff(got, tc.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestParseDNSKeyFunction_Run_Errors(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input string
	}{
		"too-few-fields": {
			input: "257 3",
		},
		"empty-string": {
			input: "",
		},
		"invalid-flags": {
			input: "abc 3 13 key==",
		},
		"invalid-protocol": {
			input: "257 abc 13 key==",
		},
		"invalid-algorithm": {
			input: "257 3 abc key==",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := function.RunResponse{
				Result: function.NewResultData(types.ObjectUnknown(dnskeyAttrTypes)),
			}
			(&parseDNSKeyFunction{}).Run(context.Background(), function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue(tc.input)}),
			}, &got)

			if got.Error == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestAccParseDNSKeyFunction_Valid(t *testing.T) {
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
  value = provider::desec::parse_dnskey("257 3 13 MemvLhocKfZ8t/7kAef1UJ3cZGjkZLM3c+e76HZ2d2z7EW+6J2EbhHWAUcHhH/JZ5PtNi0GpEy5U56WvLswZAA==")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"flags":      knownvalue.Int64Exact(257),
						"protocol":   knownvalue.Int64Exact(3),
						"algorithm":  knownvalue.Int64Exact(13),
						"public_key": knownvalue.StringExact("MemvLhocKfZ8t/7kAef1UJ3cZGjkZLM3c+e76HZ2d2z7EW+6J2EbhHWAUcHhH/JZ5PtNi0GpEy5U56WvLswZAA=="),
					})),
				},
			},
		},
	})
}

func TestAccParseDNSKeyFunction_ZSK(t *testing.T) {
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
  value = provider::desec::parse_dnskey("256 3 8 AwEAAd==")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"flags":      knownvalue.Int64Exact(256),
						"protocol":   knownvalue.Int64Exact(3),
						"algorithm":  knownvalue.Int64Exact(8),
						"public_key": knownvalue.StringExact("AwEAAd=="),
					})),
				},
			},
		},
	})
}

func TestAccParseDNSKeyFunction_FieldAccess(t *testing.T) {
	_, factories := newTestAccEnv(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
locals {
  parsed = provider::desec::parse_dnskey("257 3 13 MemvLhocKfZ8t/7kAef1UJ3cZGjkZLM3c+e76HZ2d2z7EW+6J2EbhHWAUcHhH/JZ5PtNi0GpEy5U56WvLswZAA==")
}

output "flags" {
  value = local.parsed.flags
}

output "algorithm" {
  value = local.parsed.algorithm
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("flags", knownvalue.Int64Exact(257)),
					statecheck.ExpectKnownOutputValue("algorithm", knownvalue.Int64Exact(13)),
				},
			},
		},
	})
}
