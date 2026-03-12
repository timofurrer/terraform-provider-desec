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

func TestAccZonefileDataSource(t *testing.T) {
	domainName := testAccDomainName(t, "zonefile-test")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccZonefileDataSourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_zonefile.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"data.desec_zonefile.test",
						tfjsonpath.New("zonefile"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccZonefileDataSourceConfig(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_record" "test" {
  domain = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  records = ["1.2.3.4"]
}

data "desec_zonefile" "test" {
  name = desec_domain.test.name
  depends_on = [desec_record.test]
}
`, providerConfig, name)
}
