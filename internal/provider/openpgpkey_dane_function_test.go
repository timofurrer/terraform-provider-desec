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

func mustOpenPGPKeyDANEObject(domain, subname, rdata string) types.Object {
	v, diags := types.ObjectValue(openpgpkeyDANEAttrTypes, map[string]attr.Value{
		"domain":  types.StringValue(domain),
		"subname": types.StringValue(subname),
		"type":    types.StringValue("OPENPGPKEY"),
		"rdata":   types.StringValue(rdata),
	})
	if diags.HasError() {
		panic("failed to construct OPENPGPKEY DANE object: " + diags.Errors()[0].Detail())
	}
	return v
}

func TestOpenPGPKeyDANEFunction_Run(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		request  function.RunRequest
		expected function.RunResponse
	}{
		"rfc7929-example-hugh": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("hugh@example.com"),
					types.StringValue("mQCNAzIG"),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustOpenPGPKeyDANEObject(
					"example.com",
					"c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6._openpgpkey",
					"mQCNAzIG",
				)),
			},
		},
		"uppercase-local-part": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("Hugh@example.com"),
					types.StringValue("mQCNAzIG"),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustOpenPGPKeyDANEObject(
					"example.com",
					"c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6._openpgpkey",
					"mQCNAzIG",
				)),
			},
		},
		"mixed-case-local-part": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("HuGh@example.com"),
					types.StringValue("dGVzdA=="),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustOpenPGPKeyDANEObject(
					"example.com",
					"c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6._openpgpkey",
					"dGVzdA==",
				)),
			},
		},
		"different-domain": {
			request: function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue("timo@furrer.life"),
					types.StringValue("AAAA"),
				}),
			},
			expected: function.RunResponse{
				Result: function.NewResultData(mustOpenPGPKeyDANEObject(
					"furrer.life",
					"85ee877a32d85a6178900edb730f4fad69d85d9e33ccb42f9ddd61e6._openpgpkey",
					"AAAA",
				)),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := function.RunResponse{
				Result: function.NewResultData(types.ObjectUnknown(openpgpkeyDANEAttrTypes)),
			}
			(&openpgpkeyDANEFunction{}).Run(context.Background(), tc.request, &got)

			if diff := cmp.Diff(got, tc.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestOpenPGPKeyDANEFunction_Run_Errors(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		email  string
		gpgKey string
	}{
		"no-at-sign": {
			email:  "hughexample.com",
			gpgKey: "dGVzdA==",
		},
		"empty-local-part": {
			email:  "@example.com",
			gpgKey: "dGVzdA==",
		},
		"empty-domain": {
			email:  "hugh@",
			gpgKey: "dGVzdA==",
		},
		"empty-email": {
			email:  "",
			gpgKey: "dGVzdA==",
		},
		"invalid-base64": {
			email:  "hugh@example.com",
			gpgKey: "not-valid-base64!!!",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := function.RunResponse{
				Result: function.NewResultData(types.ObjectUnknown(openpgpkeyDANEAttrTypes)),
			}
			(&openpgpkeyDANEFunction{}).Run(context.Background(), function.RunRequest{
				Arguments: function.NewArgumentsData([]attr.Value{
					types.StringValue(tc.email),
					types.StringValue(tc.gpgKey),
				}),
			}, &got)

			if got.Error == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestAccOpenPGPKeyDANEFunction_RFC7929Example(t *testing.T) {
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
  value = provider::desec::openpgpkey_dane("hugh@example.com", "mQCNAzIG")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("result", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"domain":  knownvalue.StringExact("example.com"),
						"subname": knownvalue.StringExact("c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6._openpgpkey"),
						"type":    knownvalue.StringExact("OPENPGPKEY"),
						"rdata":   knownvalue.StringExact("mQCNAzIG"),
					})),
				},
			},
		},
	})
}

func TestAccOpenPGPKeyDANEFunction_CaseInsensitive(t *testing.T) {
	_, factories := newTestAccEnv(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
output "lower" {
  value = provider::desec::openpgpkey_dane("hugh@example.com", "mQCNAzIG")
}

output "upper" {
  value = provider::desec::openpgpkey_dane("Hugh@example.com", "mQCNAzIG")
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("lower", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"domain":  knownvalue.StringExact("example.com"),
						"subname": knownvalue.StringExact("c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6._openpgpkey"),
						"type":    knownvalue.StringExact("OPENPGPKEY"),
						"rdata":   knownvalue.StringExact("mQCNAzIG"),
					})),
					statecheck.ExpectKnownOutputValue("upper", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"domain":  knownvalue.StringExact("example.com"),
						"subname": knownvalue.StringExact("c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6._openpgpkey"),
						"type":    knownvalue.StringExact("OPENPGPKEY"),
						"rdata":   knownvalue.StringExact("mQCNAzIG"),
					})),
				},
			},
		},
	})
}

func TestAccOpenPGPKeyDANEFunction_FieldAccess(t *testing.T) {
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
  dane = provider::desec::openpgpkey_dane("timo@furrer.life", "dGVzdA==")
}

output "domain" {
  value = local.dane.domain
}

output "subname" {
  value = local.dane.subname
}

output "type" {
  value = local.dane.type
}

output "rdata" {
  value = local.dane.rdata
}`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("domain", knownvalue.StringExact("furrer.life")),
					statecheck.ExpectKnownOutputValue("subname", knownvalue.StringExact("85ee877a32d85a6178900edb730f4fad69d85d9e33ccb42f9ddd61e6._openpgpkey")),
					statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("OPENPGPKEY")),
					statecheck.ExpectKnownOutputValue("rdata", knownvalue.StringExact("dGVzdA==")),
				},
			},
		},
	})
}
