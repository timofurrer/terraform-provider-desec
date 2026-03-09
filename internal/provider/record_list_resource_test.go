// Copyright (c) Timo Furrer
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

func TestAccRecordListResource(t *testing.T) {
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
				Config: testAccRecordListResourceConfig(providerConfig, domainName),
			},
			{
				Query:  true,
				Config: testAccRecordListQueryConfig(domainName),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("desec_record.all", 2),
					querycheck.ExpectIdentity("desec_record.all", map[string]knownvalue.Check{
						"domain":  knownvalue.StringExact(domainName),
						"subname": knownvalue.StringExact("@"),
						"type":    knownvalue.StringExact("A"),
					}),
					querycheck.ExpectIdentity("desec_record.all", map[string]knownvalue.Check{
						"domain":  knownvalue.StringExact(domainName),
						"subname": knownvalue.StringExact("www"),
						"type":    knownvalue.StringExact("A"),
					}),
				},
			},
		},
	})
}

func testAccRecordListResourceConfig(providerConfig, domain string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_record" "apex" {
  domain  = desec_domain.test.name
  subname = "@"
  type    = "A"
  ttl     = 3600
  records = ["1.2.3.4"]
}

resource "desec_record" "www" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  records = ["1.2.3.5"]
}
`, providerConfig, domain)
}

func testAccRecordListQueryConfig(domain string) string {
	return fmt.Sprintf(`
list "desec_record" "all" {
  provider = desec
  config {
    domain = %q
    type   = "A"
  }
}
`, domain)
}
