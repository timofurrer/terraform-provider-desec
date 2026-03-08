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

func TestAccDomainResource(t *testing.T) {
	domainName := testAccDomainName(t, "acc-test")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and Read testing.
			{
				Config: testAccDomainResourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("minimum_ttl"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("keys"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing.
			{
				ResourceName:      "desec_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDomainResourceConfig(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}
`, providerConfig, name)
}
