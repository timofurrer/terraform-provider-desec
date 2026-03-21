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

func TestAccRRsetsDataSource(t *testing.T) {
	domainName := testAccDomainName(t, "recs-ds-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRRsetsDataSourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_rrsets.all",
						tfjsonpath.New("rrsets"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
		},
	})
}

func TestAccRRsetsDataSourceFilter(t *testing.T) {
	domainName := testAccDomainName(t, "recs-filter-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRRsetsDataSourceFilterConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_rrsets.filtered",
						tfjsonpath.New("rrsets"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.desec_rrsets.filtered",
						tfjsonpath.New("rrsets").AtSliceIndex(0).AtMapKey("type"),
						knownvalue.StringExact("A"),
					),
				},
			},
		},
	})
}

func testAccRRsetsDataSourceConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_rrset" "a" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  rdata   = ["1.2.3.4"]
}

resource "desec_rrset" "aaaa" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "AAAA"
  ttl     = 3600
  rdata   = ["::1"]
}

data "desec_rrsets" "all" {
  domain     = desec_domain.test.name
  subname    = "www"
  depends_on = [desec_rrset.a, desec_rrset.aaaa]
}
`, providerConfig, domainName)
}

func testAccRRsetsDataSourceFilterConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_rrset" "a" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  rdata   = ["1.2.3.4"]
}

resource "desec_rrset" "aaaa" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "AAAA"
  ttl     = 3600
  rdata   = ["::1"]
}

data "desec_rrsets" "filtered" {
  domain     = desec_domain.test.name
  subname    = "www"
  type       = "A"
  depends_on = [desec_rrset.a, desec_rrset.aaaa]
}
`, providerConfig, domainName)
}
