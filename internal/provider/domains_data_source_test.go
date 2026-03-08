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

func TestAccDomainsDataSource(t *testing.T) {
	domainName := testAccDomainName(t, "domains-ds-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainsDataSourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					// The list should be non-empty (at least our domain exists).
					statecheck.ExpectKnownValue(
						"data.desec_domains.all",
						tfjsonpath.New("domains"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccDomainsDataSourceOwnsQname(t *testing.T) {
	domainName := testAccDomainName(t, "owns-qname-test")
	providerConfig2, factories2 := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories2,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainsDataSourceOwnsQnameConfig(providerConfig2, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_domains.filter",
						tfjsonpath.New("domains"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.desec_domains.filter",
						tfjsonpath.New("domains").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.StringExact(domainName),
					),
				},
			},
		},
	})
}

func testAccDomainsDataSourceConfig(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

data "desec_domains" "all" {
  depends_on = [desec_domain.test]
}
`, providerConfig, name)
}

func testAccDomainsDataSourceOwnsQnameConfig(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

data "desec_domains" "filter" {
  owns_qname = "www.%s"
  depends_on = [desec_domain.test]
}
`, providerConfig, name, name)
}
