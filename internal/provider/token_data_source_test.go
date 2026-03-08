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

func TestAccTokenDataSource(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenDataSourceConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("ds-test-token"),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("owner"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("is_valid"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("perm_create_domain"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("perm_delete_domain"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("perm_manage_tokens"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token.test",
						tfjsonpath.New("auto_policy"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func testAccTokenDataSourceConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name = "ds-test-token"
}

data "desec_token" "test" {
  id = desec_token.test.id
}
`, providerConfig)
}
