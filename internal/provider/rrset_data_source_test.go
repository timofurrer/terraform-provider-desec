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

func TestAccRRsetDataSource(t *testing.T) {
	domainName := testAccDomainName(t, "rec-ds-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRRsetDataSourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_rrset.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"data.desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("www"),
					),
					statecheck.ExpectKnownValue(
						"data.desec_rrset.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
					statecheck.ExpectKnownValue(
						"data.desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
					statecheck.ExpectKnownValue(
						"data.desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.desec_rrset.test",
						tfjsonpath.New("name"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccRRsetDataSourceConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_rrset" "test" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  rdata = ["1.2.3.4"]
}

data "desec_rrset" "test" {
  domain  = desec_domain.test.name
  subname = desec_rrset.test.subname
  type    = desec_rrset.test.type
}
`, providerConfig, domainName)
}
