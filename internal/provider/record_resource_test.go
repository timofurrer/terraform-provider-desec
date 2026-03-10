// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create record at zone apex using "@".
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "@", 3600, `"10.0.0.1"`),
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

// TestAccRecordResourceApexSubnameDrift is a regression test for issue #7:
// when the user writes subname = "" in their config, the provider normalises it
// to "@" before storing it in state (via normalizeSubname). On the next plan,
// the config value ("") differs from the state value ("@"), which — because
// subname has RequiresReplace — produces a persistent, unwanted destroy/recreate.
//
// The fix is to add a plan modifier that normalises "" → "@" on the planned
// value so that config and state stay in sync. Until that fix is applied, this
// test will fail at Step 2 because the framework's post-apply idempotency check
// detects a non-empty plan.
func TestAccRecordResourceApexSubnameDrift(t *testing.T) {
	domainName := testAccDomainName(t, "apex-drift-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Step 1: Create with subname="" — applies successfully.
			// The provider normalises "" to "@" internally (API and state storage),
			// but semantic equality means the configured value "" is preserved in
			// the Terraform state as-is, so no persistent diff appears on the next plan.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact(""),
					),
				},
			},
			// Step 2: Re-apply the exact same config (subname still "").
			// With the bug present: config "" != state "@" → non-empty plan →
			// RequiresReplace triggers a replace → framework idempotency check
			// fails → this test fails (demonstrating the bug).
			// With the bug fixed: semantic equality recognises "" and "@" as
			// equivalent zone-apex representations → empty plan → test passes.
			{
				Config:             testAccRecordResourceConfig(providerConfig, domainName, "", 3600, `"10.0.0.1"`),
				ExpectNonEmptyPlan: false,
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

// TestAccRecordResourceApexImport tests that importing an apex record works
// correctly with both the canonical "@" form and the empty-segment "" form.
// Step 3 (import with "domain//A") is the regression case for #8: before the
// fix, ImportState writes "" into the intermediate state, which mismatches the
// "@" that Read then stores, causing ImportStateVerify to report a failure.
func TestAccRecordResourceApexImport(t *testing.T) {
	domainName := testAccDomainName(t, "apex-import-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Step 1: Create an apex A record.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "@", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
					),
				},
			},
			// Step 2: Import using "domain/@/A" — the canonical form.
			// This should pass even without the fix because normalizeSubname("@") == "@".
			{
				ResourceName:            "desec_record.test",
				ImportState:             true,
				ImportStateId:           domainName + "/@/A",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "touched"},
			},
			// Step 3: Import using "domain//A" — the empty-segment form.
			// Before the fix, ImportState writes "" into the intermediate state;
			// Read then normalises it to "@"; ImportStateVerify sees "" != "@" and fails.
			{
				Config:                  testAccRecordResourceConfig(providerConfig, domainName, "@", 3600, `"10.0.0.1"`),
				ResourceName:            "desec_record.test",
				ImportState:             true,
				ImportStateId:           domainName + "//A",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "touched"},
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

// testAccRecordResourceTypedConfig creates a record resource with a configurable type.
func testAccRecordResourceTypedConfig(providerConfig, domainName, subname, rrtype string, ttl int, records string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_record" "test" {
  domain  = desec_domain.test.name
  subname = %q
  type    = %q
  ttl     = %d
  records = [%s]
}
`, providerConfig, domainName, subname, rrtype, ttl, records)
}

// TestAccRecordResource_UpdateRecords verifies that the records set can be
// updated multiple times and that exact record values are reflected in state.
func TestAccRecordResource_UpdateRecords(t *testing.T) {
	domainName := testAccDomainName(t, "rec-upd-rec")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create with a single record.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("1.2.3.4"),
						}),
					),
				},
			},
			// Expand to multiple records.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4", "5.6.7.8", "9.10.11.12"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetSizeExact(3),
					),
				},
			},
			// Shrink back to a single different record.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.1"),
						}),
					),
				},
			},
		},
	})
}

// TestAccRecordResource_UpdateTTL verifies that the TTL attribute can be
// updated independently of the record values.
func TestAccRecordResource_UpdateTTL(t *testing.T) {
	domainName := testAccDomainName(t, "rec-upd-ttl")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
				},
			},
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 7200, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(7200),
					),
					// Records must be unchanged.
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("1.2.3.4"),
						}),
					),
				},
			},
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 14400, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(14400),
					),
				},
			},
		},
	})
}

// TestAccRecordResource_MultiValueCreate verifies that a record with multiple
// values can be created in a single step (not just accumulated via updates).
func TestAccRecordResource_MultiValueCreate(t *testing.T) {
	domainName := testAccDomainName(t, "rec-multi-create")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4", "5.6.7.8", "9.10.11.12"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetSizeExact(3),
					),
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
				},
			},
		},
	})
}

// TestAccRecordResource_TypeChange verifies that changing the record type
// (an immutable field) triggers a destroy-and-recreate cycle. After the
// recreate, the new type is reflected correctly in state.
func TestAccRecordResource_TypeChange(t *testing.T) {
	domainName := testAccDomainName(t, "rec-type-change")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create an A record.
			{
				Config: testAccRecordResourceTypedConfig(providerConfig, domainName, "www", "A", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
				},
			},
			// Change to a TXT record — immutable field triggers destroy+recreate.
			{
				Config: testAccRecordResourceTypedConfig(providerConfig, domainName, "www", "TXT", 3600, `"\"hello world\""`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("TXT"),
					),
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("www"),
					),
				},
			},
		},
	})
}

// TestAccRecordResource_SubnameChange verifies that changing the subname
// (an immutable field) triggers a destroy-and-recreate cycle.
func TestAccRecordResource_SubnameChange(t *testing.T) {
	domainName := testAccDomainName(t, "rec-sub-change")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create record at "www".
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("www"),
					),
				},
			},
			// Change subname to "mail" — immutable field, triggers destroy+recreate.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "mail", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("mail"),
					),
				},
			},
		},
	})
}

// TestAccRecordResource_DomainChange verifies that changing the domain
// (an immutable field) triggers a destroy-and-recreate cycle.
func TestAccRecordResource_DomainChange(t *testing.T) {
	domainName1 := testAccDomainName(t, "rec-dom-a")
	domainName2 := testAccDomainName(t, "rec-dom-b")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create record in domain 1.
			{
				Config: testAccRecordTwoDomainConfig(providerConfig, domainName1, domainName2, domainName1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName1),
					),
				},
			},
			// Move record to domain 2 — immutable field, triggers destroy+recreate.
			{
				Config: testAccRecordTwoDomainConfig(providerConfig, domainName1, domainName2, domainName2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName2),
					),
				},
			},
		},
	})
}

// TestAccRecordResource_OutOfBandDelete verifies that when an RRset is deleted
// outside of Terraform, the provider detects the drift and recreates the record.
func TestAccRecordResource_OutOfBandDelete(t *testing.T) {
	domainName := testAccDomainName(t, "rec-oob-del")
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create the record.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
			// Delete the RRset via the API, then re-apply the same config.
			// The provider should detect that the RRset is gone and recreate it.
			{
				PreConfig: func() {
					if err := client.DeleteRRset(context.Background(), domainName, "www", "A"); err != nil {
						t.Fatalf("out-of-band rrset delete: %v", err)
					}
				},
				Config: testAccRecordResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("records"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
		},
	})
}

// TestAccRecordResource_ApexUpdate verifies that an apex record (subname="@")
// can be updated (TTL and records) and that the semantic equality between ""
// and "@" is maintained across the update.
func TestAccRecordResource_ApexUpdate(t *testing.T) {
	domainName := testAccDomainName(t, "apex-upd")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create apex record.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "@", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
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
			// Update TTL and add a second record — must not trigger replace.
			{
				Config: testAccRecordResourceConfig(providerConfig, domainName, "@", 7200, `"10.0.0.1", "10.0.0.2"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_record.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
					),
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

// testAccRecordTwoDomainConfig builds a config that declares two domains and
// a record in whichever domain is specified by activeDomain.
func testAccRecordTwoDomainConfig(providerConfig, domain1, domain2, activeDomain string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "domain1" {
  name = %q
}

resource "desec_domain" "domain2" {
  name = %q
}

resource "desec_record" "test" {
  domain  = %q
  subname = "www"
  type    = "A"
  ttl     = 3600
  records = ["1.2.3.4"]

  depends_on = [desec_domain.domain1, desec_domain.domain2]
}
`, providerConfig, domain1, domain2, activeDomain)
}
