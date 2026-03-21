// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

// ---- Mode A (zonefile) tests ----

func TestAccRecordsResource_zonefileBasic(t *testing.T) {
	domainName := testAccDomainName(t, "records-zf-basic")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"%s. 3600 IN A 1.2.3.4\nwww.%s. 3600 IN A 5.6.7.8\n", domainName, domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("domain"), knownvalue.StringExact(domainName)),
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("zonefile"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccRecordsResource_zonefileUpdate(t *testing.T) {
	domainName := testAccDomainName(t, "records-zf-update")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 1.2.3.4\n", domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 9.9.9.9\n%s. 3600 IN MX 10 mail.%s.\n", domainName, domainName, domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 9.9.9.9\n", domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
		},
	})
}

func TestAccRecordsResource_zonefileSemanticEquivalence(t *testing.T) {
	domainName := testAccDomainName(t, "records-zf-sem")
	providerConfig, factories := newTestAccEnv(t)

	zonefile := fmt.Sprintf("www.%s. 3600 IN A 1.2.3.4\n%s. 3600 IN MX 10 mail.%s.\n",
		domainName, domainName, domainName)

	zonefileReformatted := fmt.Sprintf("; comment\n\n%s. 3600 IN MX 10 mail.%s.\n\nwww.%s. 3600 IN A 1.2.3.4\n",
		domainName, domainName, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, zonefile),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
			{
				Config:             testAccRecordsZonefileConfig(providerConfig, domainName, zonefileReformatted),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ---- Mode B (records set) tests ----

func TestAccRecordsResource_recordsBasic(t *testing.T) {
	domainName := testAccDomainName(t, "records-rs-basic")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = ""
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["5.6.7.8"]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("domain"), knownvalue.StringExact(domainName)),
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("zonefile"), knownvalue.NotNull()),
				},
			},
		},
	})
}

func TestAccRecordsResource_recordsUpdate(t *testing.T) {
	domainName := testAccDomainName(t, "records-rs-update")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["9.9.9.9"]
    },
    {
      subname = ""
      type    = "MX"
      ttl     = 3600
      rdata = ["10 mail.`+domainName+`."]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["9.9.9.9"]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
		},
	})
}

func TestAccRecordsResource_recordsApexAt(t *testing.T) {
	domainName := testAccDomainName(t, "records-rs-apex")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "@"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
		},
	})
}

// ---- Shared tests (drift, import, coexistence) ----

func TestAccRecordsResource_drift(t *testing.T) {
	domainName := testAccDomainName(t, "records-drift")
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 1.2.3.4\nmail.%s. 3600 IN A 2.3.4.5\n", domainName, domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
			{
				PreConfig: func() {
					if err := client.DeleteRRset(t.Context(), domainName, "www", "A"); err != nil {
						t.Fatalf("out-of-band delete failed: %v", err)
					}
				},
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 1.2.3.4\nmail.%s. 3600 IN A 2.3.4.5\n", domainName, domainName)),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 1.2.3.4\nmail.%s. 3600 IN A 2.3.4.5\n", domainName, domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
		},
	})
}

func TestAccRecordsResource_import(t *testing.T) {
	domainName := testAccDomainName(t, "records-import")
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsImportSetup(providerConfig, domainName),
			},
			{
				ResourceName:            "desec_records.imported",
				ImportState:             true,
				ImportStateId:           domainName,
				ImportStateVerifyIgnore: []string{"zonefile"},
			},
			{
				PreConfig: func() {
					rs, err := client.GetRRset(t.Context(), domainName, "www", "A")
					if err != nil || rs == nil {
						t.Fatalf("expected www A record to exist after import, got err=%v", err)
					}
				},
				Config: testAccRecordsImportSetup(providerConfig, domainName),
			},
		},
	})
}

func TestAccRecordsResource_neitherZonefileNorRecords(t *testing.T) {
	domainName := testAccDomainName(t, "records-neither")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain = desec_domain.test.name

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName),
				ExpectError: regexp.MustCompile(`Exactly one of "zonefile" or "rrsets" must be specified`),
			},
		},
	})
}

func TestAccRecordsResource_exclusiveDeletesExtras(t *testing.T) {
	domainName := testAccDomainName(t, "rec-xdel")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Step 1: Create a domain with an extra record via desec_record.
			{
				Config: fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_rrset" "extra" {
  domain  = desec_domain.test.name
  subname = "extra"
  type    = "A"
  ttl     = 3600
  rdata = ["9.9.9.9"]
}
`, providerConfig, domainName),
			},
			// Step 2: Create desec_records with exclusive=true, managing only
			// www A. The pre-existing "extra" A record should be deleted.
			{
				Config: fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain    = desec_domain.test.name
  exclusive = true
  zonefile  = "www.%s. 3600 IN A 1.2.3.4\n"

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
		},
	})
}

func TestAccRecordsResource_exclusiveDrift(t *testing.T) {
	domainName := testAccDomainName(t, "rec-xdrift")
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	config := fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain    = desec_domain.test.name
  exclusive = true
  zonefile  = "www.%s. 3600 IN A 1.2.3.4\n"

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
			// Add an out-of-band record via the API.
			{
				PreConfig: func() {
					_, err := client.CreateRRset(t.Context(), domainName, api.CreateRRsetOptions{
						Subname: "rogue",
						Type:    "A",
						TTL:     3600,
						Records: []string{"6.6.6.6"},
					})
					if err != nil {
						t.Fatalf("out-of-band create failed: %v", err)
					}
				},
				// Read picks up the rogue record (exclusive mode), plan shows a diff.
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Apply deletes the rogue record.
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
		},
	})
}

func TestAccRecordsResource_exclusiveUpdateDeletesExtras(t *testing.T) {
	domainName := testAccDomainName(t, "rec-xupd")
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	configNonExclusive := fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain    = desec_domain.test.name
  exclusive = false
  zonefile  = "www.%s. 3600 IN A 1.2.3.4\n"

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName, domainName)

	configExclusive := fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain    = desec_domain.test.name
  exclusive = true
  zonefile  = "www.%s. 3600 IN A 1.2.3.4\n"

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: configNonExclusive,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
			{
				PreConfig: func() {
					_, err := client.CreateRRset(t.Context(), domainName, api.CreateRRsetOptions{
						Subname: "rogue",
						Type:    "A",
						TTL:     3600,
						Records: []string{"6.6.6.6"},
					})
					if err != nil {
						t.Fatalf("out-of-band create failed: %v", err)
					}
				},
				Config: configExclusive,
			},
			{
				Config:             configExclusive,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccRecordsResource_exclusiveSwitchOn(t *testing.T) {
	domainName := testAccDomainName(t, "rec-xswitch")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Step 1: Create with exclusive=false plus an independent desec_record.
			{
				Config: fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain    = desec_domain.test.name
  exclusive = false
  zonefile  = "www.%s. 3600 IN A 1.2.3.4\n"

  depends_on = [desec_domain.test]
}

resource "desec_rrset" "other" {
  domain  = desec_domain.test.name
  subname = "other"
  type    = "A"
  ttl     = 3600
  rdata = ["8.8.8.8"]
}
`, providerConfig, domainName, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue("desec_rrset.other",
						tfjsonpath.New("subname"), knownvalue.StringExact("other")),
				},
			},
			// Step 2: Switch to exclusive=true, remove the desec_record.
			// The "other" A record should be deleted by desec_records.
			{
				Config: fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain    = desec_domain.test.name
  exclusive = true
  zonefile  = "www.%s. 3600 IN A 1.2.3.4\n"

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
		},
	})
}

func TestAccRecordsResource_bothZonefileAndRecords(t *testing.T) {
	domainName := testAccDomainName(t, "records-both")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain   = desec_domain.test.name
  zonefile = "www.%s. 3600 IN A 1.2.3.4\n"
  rrsets = [
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
  ]

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName, domainName),
				ExpectError: regexp.MustCompile(`Only one of "zonefile" or "rrsets" may be specified`),
			},
		},
	})
}

func TestAccRecordsResource_invalidTypeLowercase(t *testing.T) {
	domainName := testAccDomainName(t, "records-val-type")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "a"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
`),
				ExpectError: regexp.MustCompile(`must be an uppercase DNS record type`),
			},
		},
	})
}

func TestAccRecordsResource_invalidTTLZero(t *testing.T) {
	domainName := testAccDomainName(t, "records-val-ttl")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 0
      rdata = ["1.2.3.4"]
    },
`),
				ExpectError: regexp.MustCompile(`must be at least 1`),
			},
		},
	})
}

func TestAccRecordsResource_duplicateSubnameType(t *testing.T) {
	domainName := testAccDomainName(t, "records-val-dup")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
    {
      subname = "www"
      type    = "A"
      ttl     = 7200
      rdata = ["5.6.7.8"]
    },
`),
				ExpectError: regexp.MustCompile(`Duplicate RRset`),
			},
		},
	})
}

func TestAccRecordsResource_switchZonefileToRecords(t *testing.T) {
	domainName := testAccDomainName(t, "records-zf2rs")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 1.2.3.4\n%s. 3600 IN MX 10 mail.%s.\n", domainName, domainName, domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
    {
      subname = ""
      type    = "MX"
      ttl     = 3600
      rdata = ["10 mail.`+domainName+`."]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("zonefile"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
    {
      subname = ""
      type    = "MX"
      ttl     = 3600
      rdata = ["10 mail.`+domainName+`."]
    },
`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccRecordsResource_switchRecordsToZonefile(t *testing.T) {
	domainName := testAccDomainName(t, "records-rs2zf")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
    {
      subname = ""
      type    = "MX"
      ttl     = 3600
      rdata = ["10 mail.`+domainName+`."]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 1.2.3.4\n%s. 3600 IN MX 10 mail.%s.\n", domainName, domainName, domainName)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("zonefile"), knownvalue.NotNull()),
				},
			},
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, fmt.Sprintf(
					"www.%s. 3600 IN A 1.2.3.4\n%s. 3600 IN MX 10 mail.%s.\n", domainName, domainName, domainName)),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccRecordsResource_zonefileComplex(t *testing.T) {
	domainName := testAccDomainName(t, "records-zf-complex")
	providerConfig, factories := newTestAccEnv(t)

	complexZonefile := fmt.Sprintf(`
; Zone file for %[1]s
; Multiple record types and subdomains

%[1]s.          3600 IN A     203.0.113.10
%[1]s.          3600 IN AAAA  2001:db8::1
%[1]s.          3600 IN MX    10 mail.%[1]s.
%[1]s.          3600 IN MX    20 mail2.%[1]s.
%[1]s.          3600 IN TXT   "v=spf1 mx ~all"
www.%[1]s.      3600 IN A     203.0.113.10
www.%[1]s.      3600 IN A     203.0.113.11
mail.%[1]s.     3600 IN A     203.0.113.20
srv.%[1]s.      3600 IN SRV   10 60 5060 sip.%[1]s.
`, domainName)

	// Same records, completely different formatting: reversed order, extra
	// whitespace, different comment placement, blank lines removed.
	complexZonefileReordered := fmt.Sprintf(`
srv.%[1]s.      3600 IN SRV   10 60 5060 sip.%[1]s.
mail.%[1]s.     3600 IN A     203.0.113.20
; web servers
www.%[1]s.      3600 IN A     203.0.113.11
www.%[1]s.      3600 IN A     203.0.113.10
%[1]s.          3600 IN TXT   "v=spf1 mx ~all"
%[1]s.          3600 IN MX    20 mail2.%[1]s.
%[1]s.          3600 IN MX    10 mail.%[1]s.
%[1]s.          3600 IN AAAA  2001:db8::1
%[1]s.          3600 IN A     203.0.113.10
`, domainName)

	// Actual content change: update the apex A record.
	complexZonefileUpdated := fmt.Sprintf(`
%[1]s.          3600 IN A     203.0.113.99
%[1]s.          3600 IN AAAA  2001:db8::1
%[1]s.          3600 IN MX    10 mail.%[1]s.
%[1]s.          3600 IN MX    20 mail2.%[1]s.
%[1]s.          3600 IN TXT   "v=spf1 mx ~all"
www.%[1]s.      3600 IN A     203.0.113.10
www.%[1]s.      3600 IN A     203.0.113.11
mail.%[1]s.     3600 IN A     203.0.113.20
srv.%[1]s.      3600 IN SRV   10 60 5060 sip.%[1]s.
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, complexZonefile),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(7)),
				},
			},
			// Reorder everything + change comments → expect no diff.
			{
				Config:             testAccRecordsZonefileConfig(providerConfig, domainName, complexZonefileReordered),
				ExpectNonEmptyPlan: false,
			},
			// Actual change: update apex A record value.
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, complexZonefileUpdated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(7)),
				},
			},
		},
	})
}

func TestAccRecordsResource_zonefileMultiValueRRset(t *testing.T) {
	domainName := testAccDomainName(t, "records-zf-multi")
	providerConfig, factories := newTestAccEnv(t)

	zonefile := fmt.Sprintf(
		"www.%[1]s. 3600 IN A 1.2.3.4\nwww.%[1]s. 3600 IN A 5.6.7.8\nwww.%[1]s. 3600 IN A 9.10.11.12\n",
		domainName)

	// Same three A records but in reversed order.
	zonefileReordered := fmt.Sprintf(
		"www.%[1]s. 3600 IN A 9.10.11.12\nwww.%[1]s. 3600 IN A 5.6.7.8\nwww.%[1]s. 3600 IN A 1.2.3.4\n",
		domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Three A lines for the same name → one RRset with 3 values.
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, zonefile),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
			// Reverse value order → expect no diff.
			{
				Config:             testAccRecordsZonefileConfig(providerConfig, domainName, zonefileReordered),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccRecordsResource_zonefileOriginAndTTLDirectives(t *testing.T) {
	domainName := testAccDomainName(t, "records-zf-origin")
	providerConfig, factories := newTestAccEnv(t)

	zonefile := fmt.Sprintf(`$ORIGIN %s.
$TTL 3600
@    IN A     1.2.3.4
www  IN A     5.6.7.8
mail IN A     9.10.11.12
@    IN MX    10 mail
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, zonefile),
				ConfigStateChecks: []statecheck.StateCheck{
					// @/A, www/A, mail/A, @/MX = 4 RRsets
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(4)),
				},
			},
		},
	})
}

func TestAccRecordsResource_zonefileSOAIgnored(t *testing.T) {
	domainName := testAccDomainName(t, "records-zf-soa")
	providerConfig, factories := newTestAccEnv(t)

	zonefile := fmt.Sprintf(`
%[1]s. 3600 IN SOA ns1.desec.io. support.desec.io. 2024010100 86400 7200 3600000 3600
%[1]s. 3600 IN NS  ns1.desec.io.
%[1]s. 3600 IN NS  ns2.desec.org.
www.%[1]s. 3600 IN A 1.2.3.4
`, domainName)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsZonefileConfig(providerConfig, domainName, zonefile),
				ConfigStateChecks: []statecheck.StateCheck{
					// SOA and apex NS are silently skipped; only www A remains.
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
		},
	})
}

func TestAccRecordsResource_recordsMultiValue(t *testing.T) {
	domainName := testAccDomainName(t, "records-rs-multi")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["5.6.7.8", "1.2.3.4", "9.10.11.12"]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
			// Reorder record values within the set → expect no diff.
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["9.10.11.12", "1.2.3.4", "5.6.7.8"]
    },
`),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccRecordsResource_recordsReorder(t *testing.T) {
	domainName := testAccDomainName(t, "records-rs-reorder")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
    {
      subname = "mail"
      type    = "A"
      ttl     = 3600
      rdata = ["2.3.4.5"]
    },
    {
      subname = ""
      type    = "MX"
      ttl     = 3600
      rdata = ["10 mail.`+domainName+`."]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(3)),
				},
			},
			// Completely reverse the order of RRset objects → expect no diff (set semantics).
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = ""
      type    = "MX"
      ttl     = 3600
      rdata = ["10 mail.`+domainName+`."]
    },
    {
      subname = "mail"
      type    = "A"
      ttl     = 3600
      rdata = ["2.3.4.5"]
    },
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
`),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccRecordsResource_recordsTTLChange(t *testing.T) {
	domainName := testAccDomainName(t, "records-rs-ttl")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, `
    {
      subname = "www"
      type    = "A"
      ttl     = 7200
      rdata = ["1.2.3.4"]
    },
`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
				},
			},
		},
	})
}

func TestAccRecordsResource_driftModeB(t *testing.T) {
	domainName := testAccDomainName(t, "records-drift-b")
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	recordsHCL := `
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      rdata = ["1.2.3.4"]
    },
    {
      subname = "mail"
      type    = "A"
      ttl     = 3600
      rdata = ["2.3.4.5"]
    },
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, recordsHCL),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
			{
				PreConfig: func() {
					if err := client.DeleteRRset(t.Context(), domainName, "www", "A"); err != nil {
						t.Fatalf("out-of-band delete failed: %v", err)
					}
				},
				Config:             testAccRecordsSetConfig(providerConfig, domainName, recordsHCL),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccRecordsSetConfig(providerConfig, domainName, recordsHCL),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(2)),
				},
			},
		},
	})
}

func TestAccRecordsResource_coexistWithRecord(t *testing.T) {
	domainName := testAccDomainName(t, "records-coexist")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordsCoexistConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_records.test",
						tfjsonpath.New("rrsets"), knownvalue.SetSizeExact(1)),
					statecheck.ExpectKnownValue("desec_rrset.mail",
						tfjsonpath.New("subname"), knownvalue.StringExact("mail")),
				},
			},
		},
	})
}

// ---- Config helpers ----

func testAccRecordsZonefileConfig(providerConfig, domainName, zonefile string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain   = desec_domain.test.name
  zonefile = %q

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName, zonefile)
}

func testAccRecordsSetConfig(providerConfig, domainName, rrsetsHCL string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain = desec_domain.test.name

  rrsets = [
%s
  ]

  depends_on = [desec_domain.test]
}
`, providerConfig, domainName, rrsetsHCL)
}

func testAccRecordsImportSetup(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_rrset" "www" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  rdata = ["1.2.3.4"]
}

resource "desec_records" "imported" {
  domain   = desec_domain.test.name
  zonefile = "www.%s. 3600 IN A 1.2.3.4\n"

  depends_on = [desec_rrset.www]
}
`, providerConfig, domainName, domainName)
}

func testAccRecordsCoexistConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_records" "test" {
  domain   = desec_domain.test.name
  zonefile = "www.%s. 3600 IN A 1.2.3.4\n"

  depends_on = [desec_domain.test]
}

resource "desec_rrset" "mail" {
  domain  = desec_domain.test.name
  subname = "mail"
  type    = "A"
  ttl     = 3600
  rdata = ["2.3.4.5"]
}
`, providerConfig, domainName, domainName)
}

// ---- Unit tests ----

func TestParseZonefile_basic(t *testing.T) {
	domain := "example.com"
	zonefile := "example.com. 3600 IN A 1.2.3.4\nwww.example.com. 3600 IN A 5.6.7.8\nexample.com. 3600 IN MX 10 mail.example.com.\n"
	rrsets, err := parseZonefile(zonefile, domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rrsets) != 3 {
		t.Fatalf("expected 3 rrsets, got %d", len(rrsets))
	}
	cases := []struct{ subname, rrtype string }{
		{"", "A"},
		{"", "MX"},
		{"www", "A"},
	}
	for i, c := range cases {
		if rrsets[i].Subname != c.subname || rrsets[i].Type != c.rrtype {
			t.Errorf("rrsets[%d]: want (%q, %q), got (%q, %q)",
				i, c.subname, c.rrtype, rrsets[i].Subname, rrsets[i].Type)
		}
	}
}

func TestParseZonefile_skipsAutoManaged(t *testing.T) {
	domain := "example.com"
	zonefile := "example.com. 3600 IN SOA ns1.example.com. admin.example.com. 1 3600 900 604800 300\nexample.com. 3600 IN NS ns1.example.com.\nwww.example.com. 3600 IN A 1.2.3.4\n"
	rrsets, err := parseZonefile(zonefile, domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rrsets) != 1 {
		t.Fatalf("expected 1 rrset, got %d: %+v", len(rrsets), rrsets)
	}
	if rrsets[0].Subname != "www" || rrsets[0].Type != "A" {
		t.Errorf("unexpected rrset: %+v", rrsets[0])
	}
}

func TestParseZonefile_skipsOutsideDomain(t *testing.T) {
	domain := "example.com"
	zonefile := "www.example.com. 3600 IN A 1.2.3.4\nother.net. 3600 IN A 9.9.9.9\n"
	rrsets, err := parseZonefile(zonefile, domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rrsets) != 1 {
		t.Fatalf("expected 1 rrset (out-of-domain record skipped), got %d", len(rrsets))
	}
}

func TestParseZonefile_origin(t *testing.T) {
	domain := "example.com"
	zonefile := "$ORIGIN example.com.\n$TTL 3600\n@ IN A 1.2.3.4\nwww IN A 5.6.7.8\n"
	rrsets, err := parseZonefile(zonefile, domain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rrsets) != 2 {
		t.Fatalf("expected 2 rrsets, got %d: %+v", len(rrsets), rrsets)
	}
	if rrsets[0].Subname != "" || rrsets[0].Type != "A" {
		t.Errorf("expected apex A, got %+v", rrsets[0])
	}
	if rrsets[1].Subname != "www" || rrsets[1].Type != "A" {
		t.Errorf("expected www A, got %+v", rrsets[1])
	}
}

func TestRRsetSetsEqual(t *testing.T) {
	a := []api.RRset{
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4", "5.6.7.8"}},
		{Subname: "", Type: "MX", TTL: 3600, Records: []string{"10 mail.example.com."}},
	}
	b := []api.RRset{
		{Subname: "", Type: "MX", TTL: 3600, Records: []string{"10 mail.example.com."}},
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"5.6.7.8", "1.2.3.4"}},
	}
	if !rrsetSetsEqual(a, b) {
		t.Error("expected equal, got not equal")
	}

	c := []api.RRset{
		{Subname: "www", Type: "A", TTL: 7200, Records: []string{"1.2.3.4", "5.6.7.8"}},
		{Subname: "", Type: "MX", TTL: 3600, Records: []string{"10 mail.example.com."}},
	}
	if rrsetSetsEqual(a, c) {
		t.Error("expected not equal (TTL differs), got equal")
	}

	d := []api.RRset{
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
		{Subname: "", Type: "MX", TTL: 3600, Records: []string{"10 mail.example.com."}},
	}
	if rrsetSetsEqual(a, d) {
		t.Error("expected not equal (record missing), got equal")
	}
}

func TestRRsetToZonefile(t *testing.T) {
	rrsets := []api.RRset{
		{Subname: "", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
		{Subname: "www", Type: "A", TTL: 300, Records: []string{"5.6.7.8", "9.0.1.2"}},
	}
	got := rrsetToZonefile("example.com", rrsets)
	want := "example.com.\t3600\tIN\tA\t1.2.3.4\n" +
		"www.example.com.\t300\tIN\tA\t5.6.7.8\n" +
		"www.example.com.\t300\tIN\tA\t9.0.1.2\n"
	if got != want {
		t.Errorf("rrsetToZonefile:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSetStateAfterWrite_recordsMode(t *testing.T) {
	ctx := t.Context()
	r := &recordsResource{}

	planRecords := []api.RRset{
		{Subname: "www", Type: "A", TTL: 60, Records: []string{"1.2.3.4"}},
	}
	planSet, diags := apiRRsetsToSet(ctx, planRecords)
	if diags.HasError() {
		t.Fatalf("apiRRsetsToSet: %v", diags)
	}

	data := &recordsResourceModel{
		Domain:    types.StringValue("example.com"),
		Exclusive: types.BoolValue(false),
		Zonefile:  types.StringNull(),
		RRsets:    planSet,
	}

	returned := []api.RRset{
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
	}

	diags = r.setStateAfterWrite(ctx, data, "example.com", returned)
	if diags.HasError() {
		t.Fatalf("setStateAfterWrite: %v", diags)
	}

	gotRRsets, diags := recordsSetToAPIRRsets(ctx, data.RRsets)
	if diags.HasError() {
		t.Fatalf("recordsSetToAPIRRsets: %v", diags)
	}

	if len(gotRRsets) != 1 {
		t.Fatalf("expected 1 rrset, got %d", len(gotRRsets))
	}
	if gotRRsets[0].TTL != 3600 {
		t.Errorf("expected TTL 3600 (server-normalized), got %d", gotRRsets[0].TTL)
	}
}
