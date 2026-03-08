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
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccRecordResource(t *testing.T) {
	domainName := testAccDomainName(t, "rec-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and Read.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("www"),
					),
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
			// ImportState testing.
			{
				ResourceName:      "desec_record.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/www/A", domainName),
			},
			// Update TTL and records.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 7200, `"1.2.3.4", "5.6.7.8"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(7200),
					),
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetSizeExact(2),
					),
				},
			},
		},
	})
}

func TestAccRecordResourceApex(t *testing.T) {
	domainName := testAccDomainName(t, "apex-rec-acc")
	providerConfig2, factories2 := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories2,
		Steps: []resource.TestStep{
			// Create record at zone apex using "@".
			{
				Config: testAccRecordResourceConfig(providerConfig2, domainName, "@", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
					),
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
				},
			},
		},
	})
}

func TestAccRecordResourceIdentity(t *testing.T) {
	domainName := testAccDomainName(t, "id-rec-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			// Create and verify identity is set.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						"desec_record.test",
						map[string]knownvalue.Check{
							"domain":  knownvalue.StringExact(domainName),
							"subname": knownvalue.StringExact("www"),
							"type":    knownvalue.StringExact("A"),
						},
					),
				},
			},
			// Import using identity.
			{
				ResourceName:    "desec_record.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccRecordResourceConfig(providerConfig, domainName, subname string, ttl int, records string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_record" "test" {
  domain  = desec_domain.test.name
  subname = %q
  type    = "A"
  ttl     = %d
  records = [%s]
}
`, providerConfig, domainName, subname, ttl, records)
}
