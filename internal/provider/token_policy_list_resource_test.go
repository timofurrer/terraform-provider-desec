// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
)

func TestAccTokenPolicyListResource(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create token + policies, then query the list resource.
			{
				Config: testAccTokenPolicyListResourceConfig(providerConfig),
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

func testAccTokenPolicyListResourceConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name = "policy-list-test"
}

resource "desec_token_policy" "default" {
  token_id = desec_token.test.id
}

resource "desec_token_policy" "specific" {
  token_id   = desec_token.test.id
  domain     = "example.com"
  perm_write = true
  depends_on = [desec_token_policy.default]
}
`, providerConfig)
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
