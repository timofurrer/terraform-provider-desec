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

func TestAccTokenListResource(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create two tokens, then query the list resource.
			{
				Config: testAccTokenListResourceConfig(providerConfig),
			},
			{
				Query:  true,
				Config: testAccTokenListQueryConfig(providerConfig),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("desec_token.all", 2),
					querycheck.ExpectIdentity("desec_token.all", map[string]knownvalue.Check{
						"id": knownvalue.NotNull(),
					}),
				},
			},
		},
	})
}

func testAccTokenListResourceConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "one" {
  name = "test-token-one"
}

resource "desec_token" "two" {
  name = "test-token-two"
}
`, providerConfig)
}

func testAccTokenListQueryConfig(providerConfig string) string {
	return `
list "desec_token" "all" {
  provider = desec
}
`
}
