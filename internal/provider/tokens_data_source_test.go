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

func TestAccTokensDataSource(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokensDataSourceConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					// Do not assert an exact list size: a real account may have
					// additional tokens (e.g. the bootstrap auth token). Instead,
					// verify that both created tokens appear somewhere in the list.
					tokenListContains("data.desec_tokens.all", "token-alpha"),
					tokenListContains("data.desec_tokens.all", "token-beta"),
					// Spot-check structural attributes on the first element.
					statecheck.ExpectKnownValue(
						"data.desec_tokens.all",
						tfjsonpath.New("tokens").AtSliceIndex(0).AtMapKey("is_valid"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.desec_tokens.all",
						tfjsonpath.New("tokens").AtSliceIndex(0).AtMapKey("allowed_subnets"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccTokensDataSourceConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "alpha" {
  name = "token-alpha"
}

resource "desec_token" "beta" {
  name       = "token-beta"
  depends_on = [desec_token.alpha]
}

data "desec_tokens" "all" {
  depends_on = [desec_token.alpha, desec_token.beta]
}
`, providerConfig)
}
