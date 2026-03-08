// Copyright (c) Timo Furrer
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
					statecheck.ExpectKnownValue(
						"data.desec_tokens.all",
						tfjsonpath.New("tokens"),
						knownvalue.ListSizeExact(2),
					),
					// First token object has the expected name.
					statecheck.ExpectKnownValue(
						"data.desec_tokens.all",
						tfjsonpath.New("tokens").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.StringExact("token-alpha"),
					),
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
					// Second token.
					statecheck.ExpectKnownValue(
						"data.desec_tokens.all",
						tfjsonpath.New("tokens").AtSliceIndex(1).AtMapKey("name"),
						knownvalue.StringExact("token-beta"),
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
