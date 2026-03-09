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

func TestAccDomainListResource(t *testing.T) {
	domainName1 := testAccDomainName(t, "list-1")
	domainName2 := testAccDomainName(t, "list-2")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			// Create two domains, then query the list resource.
			{
				Config:            testAccDomainListResourceConfig(providerConfig, domainName1, domainName2),
				ConfigStateChecks: nil,
			},
			{
				Query:  true,
				Config: testAccDomainListQueryConfig(providerConfig),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("desec_domain.all", 2),
					querycheck.ExpectIdentity("desec_domain.all", map[string]knownvalue.Check{
						"name": knownvalue.StringExact(domainName1),
					}),
					querycheck.ExpectIdentity("desec_domain.all", map[string]knownvalue.Check{
						"name": knownvalue.StringExact(domainName2),
					}),
				},
			},
		},
	})
}

func testAccDomainListResourceConfig(providerConfig, name1, name2 string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "one" {
  name = %q
}

resource "desec_domain" "two" {
  name = %q
}
`, providerConfig, name1, name2)
}

func testAccDomainListQueryConfig(providerConfig string) string {
	return `
list "desec_domain" "all" {
  provider = desec
}
`
}
