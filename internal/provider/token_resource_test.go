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
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccTokenResource(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and Read testing.
			{
				Config: testAccTokenResourceConfig(providerConfig, "my-token", true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("my-token"),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("token"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("owner"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("is_valid"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_create_domain"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_delete_domain"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_manage_tokens"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing — token secret cannot be recovered, so ignore it.
			{
				ResourceName:            "desec_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update: change name and permissions.
			{
				Config: testAccTokenResourceConfig(providerConfig, "updated-token", false, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("updated-token"),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_create_domain"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_delete_domain"),
						knownvalue.Bool(true),
					),
					// Token secret must still be present in state after update.
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("token"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccTokenResourceWithSubnets(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenResourceWithSubnetsConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("192.168.0.0/24"),
						}),
					),
				},
			},
		},
	})
}

func TestAccTokenResourceIdentity(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			// Create and verify identity is set.
			{
				Config: testAccTokenResourceConfig(providerConfig, "identity-token", true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentityValue(
						"desec_token.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Import using identity.
			{
				ResourceName:    "desec_token.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTokenResourceConfig(providerConfig, name string, permCreate, permDelete bool) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name               = %q
  perm_create_domain = %t
  perm_delete_domain = %t
}
`, providerConfig, name, permCreate, permDelete)
}

func testAccTokenResourceWithSubnetsConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name            = "subnet-token"
  allowed_subnets = ["192.168.0.0/24"]
}
`, providerConfig)
}
