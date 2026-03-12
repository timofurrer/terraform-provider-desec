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

func TestAccDomainDataSource(t *testing.T) {
	domainName := testAccDomainName(t, "ds-acc-test")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainDataSourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"data.desec_domain.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"data.desec_domain.test",
						tfjsonpath.New("minimum_ttl"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.desec_domain.test",
						tfjsonpath.New("keys"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccDomainDataSourceConfig(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

data "desec_domain" "test" {
  name = desec_domain.test.name
}
`, providerConfig, name)
}
