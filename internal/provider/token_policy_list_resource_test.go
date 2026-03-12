// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccTokenPolicyListResource(t *testing.T) {
	domainName := testAccDomainName(t, "policy-list")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			// Create token + policies, then query the list resource.
			{
				Config: testAccTokenPolicyListResourceConfig(providerConfig, domainName),
			},
			{
				Query:  true,
				Config: testAccTokenPolicyListQueryConfig(providerConfig),
				QueryResultChecks: []querycheck.QueryResultCheck{
					// default policy + one specific policy = 2
					querycheck.ExpectLength("desec_token_policy.all", 2),
					querycheck.ExpectIdentity("desec_token_policy.all", map[string]knownvalue.Check{
						"token_id": knownvalue.NotNull(),
						"id":       knownvalue.NotNull(),
					}),
				},
			},
		},
	})
}

func testAccTokenPolicyListResourceConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_token" "test" {
  name = "policy-list-test"
}

resource "desec_token_policy" "default" {
  token_id = desec_token.test.id
}

resource "desec_token_policy" "specific" {
  token_id   = desec_token.test.id
  domain     = desec_domain.test.name
  perm_write = true
  depends_on = [desec_token_policy.default]
}
`, providerConfig, domainName)
}

func testAccTokenPolicyListQueryConfig(providerConfig string) string {
	return `
list "desec_token_policy" "all" {
  provider = desec
  config {
    token_id = desec_token.test.id
  }
}
`
}
