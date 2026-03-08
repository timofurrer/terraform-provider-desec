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

func TestAccRecordsDataSource(t *testing.T) {
	domainName := testAccDomainName(t, "recs-ds-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsDataSourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_records.all",
						tfjsonpath.New("records"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
		},
	})
}

func TestAccRecordsDataSourceFilter(t *testing.T) {
	domainName := testAccDomainName(t, "recs-filter-acc")
	providerConfig2, factories2 := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories2,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsDataSourceFilterConfig(providerConfig2, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_records.filtered",
						tfjsonpath.New("records"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.desec_records.filtered",
						tfjsonpath.New("records").AtSliceIndex(0).AtMapKey("type"),
						knownvalue.StringExact("A"),
					),
				},
			},
		},
	})
}

func testAccRecordsDataSourceConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_record" "a" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  records = ["1.2.3.4"]
}

resource "desec_record" "aaaa" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "AAAA"
  ttl     = 3600
  records = ["::1"]
}

data "desec_records" "all" {
  domain     = desec_domain.test.name
  subname    = "www"
  depends_on = [desec_record.a, desec_record.aaaa]
}
`, providerConfig, domainName)
}

func testAccRecordsDataSourceFilterConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_record" "a" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  records = ["1.2.3.4"]
}

resource "desec_record" "aaaa" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "AAAA"
  ttl     = 3600
  records = ["::1"]
}

data "desec_records" "filtered" {
  domain     = desec_domain.test.name
  subname    = "www"
  type       = "A"
  depends_on = [desec_record.a, desec_record.aaaa]
}
`, providerConfig, domainName)
}
