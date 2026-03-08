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

func TestAccTokenPolicyDataSource(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenPolicyDataSourceConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_token_policy.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token_policy.test",
						tfjsonpath.New("token_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token_policy.test",
						tfjsonpath.New("domain"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token_policy.test",
						tfjsonpath.New("subname"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token_policy.test",
						tfjsonpath.New("type"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_token_policy.test",
						tfjsonpath.New("perm_write"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func testAccTokenPolicyDataSourceConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name = "ds-policy-token"
}

resource "desec_token_policy" "default" {
  token_id = desec_token.test.id
}

data "desec_token_policy" "test" {
  token_id = desec_token.test.id
  id       = desec_token_policy.default.id
}
`, providerConfig)
}
