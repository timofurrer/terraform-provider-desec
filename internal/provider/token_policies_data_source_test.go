// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTokenPoliciesDataSource(t *testing.T) {
	domainName := testAccDomainName(t, "policies-ds-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenPoliciesDataSourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					// Two policies: default + one specific.
					statecheck.ExpectKnownValue(
						"data.desec_token_policies.test",
						tfjsonpath.New("policies"),
						knownvalue.ListSizeExact(2),
					),
					// First entry is the default policy (sorted first by fake server).
					statecheck.ExpectKnownValue(
						"data.desec_token_policies.test",
						tfjsonpath.New("policies").AtSliceIndex(0).AtMapKey("domain"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token_policies.test",
						tfjsonpath.New("policies").AtSliceIndex(0).AtMapKey("perm_write"),
						knownvalue.Bool(false),
					),
					// Second entry is the specific policy.
					statecheck.ExpectKnownValue(
						"data.desec_token_policies.test",
						tfjsonpath.New("policies").AtSliceIndex(1).AtMapKey("domain"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token_policies.test",
						tfjsonpath.New("policies").AtSliceIndex(1).AtMapKey("perm_write"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

func testAccTokenPoliciesDataSourceConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_token" "test" {
  name = "policies-ds-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = false
}

resource "desec_token_policy" "specific" {
  token_id   = desec_token.test.id
  domain     = desec_domain.test.name
  perm_write = true

  depends_on = [desec_token_policy.default]
}

data "desec_token_policies" "test" {
  token_id = desec_token.test.id

  depends_on = [desec_token_policy.default, desec_token_policy.specific]
}
`, providerConfig, domainName)
}
