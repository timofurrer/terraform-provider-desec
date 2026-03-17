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

func mustDSObject(keyTag, algorithm, digestType int64, digest string) types.Object {
	v, diags := types.ObjectValue(dsAttrTypes, map[string]attr.Value{
		"key_tag":     types.Int64Value(keyTag),
		"algorithm":   types.Int64Value(algorithm),
		"digest_type": types.Int64Value(digestType),
		"digest":      types.StringValue(digest),
	})
	if diags.HasError() {
		panic("failed to construct DS object: " + diags.Errors()[0].Detail())
	}
	return v
}

func TestParseDSFunction_Run(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		request  function.RunRequest
		expected function.RunResponse
	}{
		"sha256": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("26064 13 2 ed8e5b1f29bb64d404e1c43ff7d15232289eb707e554808309bbd5f7fb4695d0"),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustDSObject(26064, 13, 2, "ed8e5b1f29bb64d404e1c43ff7d15232289eb707e554808309bbd5f7fb4695d0")),
			},
		},
		"sha384": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("26064 13 4 c846b8331e1912663653ea67d9a6680ab850530f483003c36d90849e0fc4d4f3310d91ab2358823b75b001e7012b749e"),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustDSObject(26064, 13, 4, "c846b8331e1912663653ea67d9a6680ab850530f483003c36d90849e0fc4d4f3310d91ab2358823b75b001e7012b749e")),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := function.RunResponse{
				Result: function.NewResultData(types.ObjectUnknown(dsAttrTypes)),
			}
			(&parseDSFunction{}).Run(context.Background(), tc.request, &got)

			if diff := cmp.Diff(got, tc.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestParseDSFunction_Run_Errors(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input string
	}{
		"too-few-fields": {
			input: "26064 13",
		},
		"empty-string": {
			input: "",
		},
		"invalid-key-tag": {
			input: "abc 13 2 digest",
		},
		"invalid-algorithm": {
			input: "26064 abc 2 digest",
		},
		"invalid-digest-type": {
			input: "26064 13 abc digest",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := function.RunResponse{
				Result: function.NewResultData(types.ObjectUnknown(dsAttrTypes)),
			}
			(&parseDSFunction{}).Run(context.Background(), function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{types.StringValue(tc.input)}),
			}, &got)

			if got.Error == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestAccParseDSFunction_SHA256(t *testing.T) {
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
  value = provider::desec::parse_ds("26064 13 2 ed8e5b1f29bb64d404e1c43ff7d15232289eb707e554808309bbd5f7fb4695d0")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"key_tag":     knownvalue.Int64Exact(26064),
						"algorithm":   knownvalue.Int64Exact(13),
						"digest_type": knownvalue.Int64Exact(2),
						"digest":      knownvalue.StringExact("ed8e5b1f29bb64d404e1c43ff7d15232289eb707e554808309bbd5f7fb4695d0"),
					})),
				},
			},
		},
	})
}

func TestAccParseDSFunction_SHA384(t *testing.T) {
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
  value = provider::desec::parse_ds("26064 13 4 c846b8331e1912663653ea67d9a6680ab850530f483003c36d90849e0fc4d4f3310d91ab2358823b75b001e7012b749e")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"key_tag":     knownvalue.Int64Exact(26064),
						"algorithm":   knownvalue.Int64Exact(13),
						"digest_type": knownvalue.Int64Exact(4),
						"digest":      knownvalue.StringExact("c846b8331e1912663653ea67d9a6680ab850530f483003c36d90849e0fc4d4f3310d91ab2358823b75b001e7012b749e"),
					})),
				},
			},
		},
	})
}

func TestAccParseDSFunction_FieldAccess(t *testing.T) {
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
  parsed = provider::desec::parse_ds("26064 13 2 ed8e5b1f29bb64d404e1c43ff7d15232289eb707e554808309bbd5f7fb4695d0")
}

output "key_tag" {
  value = local.parsed.key_tag
}

output "digest" {
  value = local.parsed.digest
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("key_tag", knownvalue.Int64Exact(26064)),
					statecheck.ExpectKnownOutputValue("digest", knownvalue.StringExact("ed8e5b1f29bb64d404e1c43ff7d15232289eb707e554808309bbd5f7fb4695d0")),
				},
			},
		},
	})
}
