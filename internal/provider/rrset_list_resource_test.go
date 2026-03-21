// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccRRsetListResource(t *testing.T) {
	domainName := testAccDomainName(t, "record-list")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			// Create domain + two records, then query the list resource.
			{
				Config: testAccRRsetListResourceConfig(providerConfig, domainName),
			},
			{
				Query:  true,
				Config: testAccRRsetListQueryConfig(domainName),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("desec_rrset.all", 2),
					querycheck.ExpectIdentity("desec_rrset.all", map[string]knownvalue.Check{
						"domain":  knownvalue.StringExact(domainName),
						"subname": knownvalue.StringExact("@"),
						"type":    knownvalue.StringExact("A"),
					}),
					querycheck.ExpectIdentity("desec_rrset.all", map[string]knownvalue.Check{
						"domain":  knownvalue.StringExact(domainName),
						"subname": knownvalue.StringExact("www"),
						"type":    knownvalue.StringExact("A"),
					}),
				},
			},
		},
	})
}

func testAccRRsetListResourceConfig(providerConfig, domain string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_rrset" "apex" {
  domain  = desec_domain.test.name
  subname = "@"
  type    = "A"
  ttl     = 3600
  rdata = ["1.2.3.4"]
}

resource "desec_rrset" "www" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  rdata = ["1.2.3.5"]
}
`, providerConfig, domain)
}

func testAccRRsetListQueryConfig(domain string) string {
	return fmt.Sprintf(`
list "desec_rrset" "all" {
  provider = desec
  config {
    domain = %q
    type   = "A"
  }
}
`, domain)
}
