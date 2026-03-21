// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccRRsetResource(t *testing.T) {
	domainName := testAccDomainName(t, "rec-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and Read.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("www"),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
			// ImportState testing.
			{
				ResourceName:      "desec_rrset.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     fmt.Sprintf("%s/www/A", domainName),
			},
			// Update TTL and records.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 7200, `"1.2.3.4", "5.6.7.8"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(7200),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetSizeExact(2),
					),
				},
			},
		},
	})
}

func TestAccRRsetResourceApex(t *testing.T) {
	domainName := testAccDomainName(t, "apex-rec-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create record at zone apex using "@".
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "@", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
				},
			},
		},
	})
}

// TestAccRRsetResourceApexSubnameDrift is a regression test for issue #7:
// when the user writes subname = "" in their config, the provider normalises it
// to "@" before storing it in state (via normalizeSubname). On the next plan,
// the config value ("") differs from the state value ("@"), which — because
// subname has RequiresReplace — produces a persistent, unwanted destroy/recreate.
//
// The fix is to add a plan modifier that normalises "" → "@" on the planned
// value so that config and state stay in sync. Until that fix is applied, this
// test will fail at Step 2 because the framework's post-apply idempotency check
// detects a non-empty plan.
func TestAccRRsetResourceApexSubnameDrift(t *testing.T) {
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
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
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
				Config:             testAccRRsetResourceConfig(providerConfig, domainName, "", 3600, `"10.0.0.1"`),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccRRsetResourceIdentity(t *testing.T) {
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
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						"desec_rrset.test",
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
				ResourceName:    "desec_rrset.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// TestAccRRsetResourceApexImport tests that importing an apex record works
// correctly with both the canonical "@" form and the empty-segment "" form.
// Step 3 (import with "domain//A") is the regression case for #8: before the
// fix, ImportState writes "" into the intermediate state, which mismatches the
// "@" that Read then stores, causing ImportStateVerify to report a failure.
func TestAccRRsetResourceApexImport(t *testing.T) {
	domainName := testAccDomainName(t, "apex-import-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Step 1: Create an apex A record.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "@", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
					),
				},
			},
			// Step 2: Import using "domain/@/A" — the canonical form.
			// This should pass even without the fix because normalizeSubname("@") == "@".
			{
				ResourceName:            "desec_rrset.test",
				ImportState:             true,
				ImportStateId:           domainName + "/@/A",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "touched"},
			},
			// Step 3: Import using "domain//A" — the empty-segment form.
			// Before the fix, ImportState writes "" into the intermediate state;
			// Read then normalises it to "@"; ImportStateVerify sees "" != "@" and fails.
			{
				Config:                  testAccRRsetResourceConfig(providerConfig, domainName, "@", 3600, `"10.0.0.1"`),
				ResourceName:            "desec_rrset.test",
				ImportState:             true,
				ImportStateId:           domainName + "//A",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"created", "touched"},
			},
		},
	})
}

func testAccRRsetResourceConfig(providerConfig, domainName, subname string, ttl int, records string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_rrset" "test" {
  domain  = desec_domain.test.name
  subname = %q
  type    = "A"
  ttl     = %d
  rdata = [%s]
}
`, providerConfig, domainName, subname, ttl, records)
}

// testAccRRsetResourceTypedConfig creates a record resource with a configurable type.
func testAccRRsetResourceTypedConfig(providerConfig, domainName, subname, rrtype string, ttl int, records string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_rrset" "test" {
  domain  = desec_domain.test.name
  subname = %q
  type    = %q
  ttl     = %d
  rdata = [%s]
}
`, providerConfig, domainName, subname, rrtype, ttl, records)
}

// TestAccRRsetResource_UpdateRecords verifies that the records set can be
// updated multiple times and that exact record values are reflected in state.
func TestAccRRsetResource_UpdateRecords(t *testing.T) {
	domainName := testAccDomainName(t, "rec-upd-rec")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create with a single record.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("1.2.3.4"),
						}),
					),
				},
			},
			// Expand to multiple records.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4", "5.6.7.8", "9.10.11.12"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetSizeExact(3),
					),
				},
			},
			// Shrink back to a single different record.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.1"),
						}),
					),
				},
			},
		},
	})
}

// TestAccRRsetResource_UpdateTTL verifies that the TTL attribute can be
// updated independently of the record values.
func TestAccRRsetResource_UpdateTTL(t *testing.T) {
	domainName := testAccDomainName(t, "rec-upd-ttl")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
				},
			},
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 7200, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(7200),
					),
					// Records must be unchanged.
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("1.2.3.4"),
						}),
					),
				},
			},
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 14400, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(14400),
					),
				},
			},
		},
	})
}

// TestAccRRsetResource_MultiValueCreate verifies that a record with multiple
// values can be created in a single step (not just accumulated via updates).
func TestAccRRsetResource_MultiValueCreate(t *testing.T) {
	domainName := testAccDomainName(t, "rec-multi-create")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4", "5.6.7.8", "9.10.11.12"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetSizeExact(3),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
				},
			},
		},
	})
}

// TestAccRRsetResource_TypeChange verifies that changing the record type
// (an immutable field) triggers a destroy-and-recreate cycle. After the
// recreate, the new type is reflected correctly in state.
func TestAccRRsetResource_TypeChange(t *testing.T) {
	domainName := testAccDomainName(t, "rec-type-change")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create an A record.
			{
				Config: testAccRRsetResourceTypedConfig(providerConfig, domainName, "www", "A", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
				},
			},
			// Change to a TXT record — immutable field triggers destroy+recreate.
			{
				Config: testAccRRsetResourceTypedConfig(providerConfig, domainName, "www", "TXT", 3600, `"\"hello world\""`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("TXT"),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("www"),
					),
				},
			},
		},
	})
}

// TestAccRRsetResource_SubnameChange verifies that changing the subname
// (an immutable field) triggers a destroy-and-recreate cycle.
func TestAccRRsetResource_SubnameChange(t *testing.T) {
	domainName := testAccDomainName(t, "rec-sub-change")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create record at "www".
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("www"),
					),
				},
			},
			// Change subname to "mail" — immutable field, triggers destroy+recreate.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "mail", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("mail"),
					),
				},
			},
		},
	})
}

// TestAccRRsetResource_DomainChange verifies that changing the domain
// (an immutable field) triggers a destroy-and-recreate cycle.
func TestAccRRsetResource_DomainChange(t *testing.T) {
	domainName1 := testAccDomainName(t, "rec-dom-a")
	domainName2 := testAccDomainName(t, "rec-dom-b")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create record in domain 1.
			{
				Config: testAccRRsetTwoDomainConfig(providerConfig, domainName1, domainName2, domainName1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName1),
					),
				},
			},
			// Move record to domain 2 — immutable field, triggers destroy+recreate.
			{
				Config: testAccRRsetTwoDomainConfig(providerConfig, domainName1, domainName2, domainName2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName2),
					),
				},
			},
		},
	})
}

// TestAccRRsetResource_OutOfBandDelete verifies that when an RRset is deleted
// outside of Terraform, the provider detects the drift and recreates the record.
func TestAccRRsetResource_OutOfBandDelete(t *testing.T) {
	domainName := testAccDomainName(t, "rec-oob-del")
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create the record.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
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
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "www", 3600, `"1.2.3.4"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
		},
	})
}

// TestAccRRsetResource_ApexUpdate verifies that an apex record (subname="@")
// can be updated (TTL and records) and that the semantic equality between ""
// and "@" is maintained across the update.
func TestAccRRsetResource_ApexUpdate(t *testing.T) {
	domainName := testAccDomainName(t, "apex-upd")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create apex record.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "@", 3600, `"10.0.0.1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetSizeExact(1),
					),
				},
			},
			// Update TTL and add a second record — must not trigger replace.
			{
				Config: testAccRRsetResourceConfig(providerConfig, domainName, "@", 7200, `"10.0.0.1", "10.0.0.2"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(7200),
					),
					statecheck.ExpectKnownValue(
						"desec_rrset.test",
						tfjsonpath.New("rdata"),
						knownvalue.SetSizeExact(2),
					),
				},
			},
		},
	})
}

// testAccRRsetTwoDomainConfig builds a config that declares two domains and
// a record in whichever domain is specified by activeDomain.
func testAccRRsetTwoDomainConfig(providerConfig, domain1, domain2, activeDomain string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "domain1" {
  name = %q
}

resource "desec_domain" "domain2" {
  name = %q
}

resource "desec_rrset" "test" {
  domain  = %q
  subname = "www"
  type    = "A"
  ttl     = 3600
  rdata = ["1.2.3.4"]

  depends_on = [desec_domain.domain1, desec_domain.domain2]
}
`, providerConfig, domain1, domain2, activeDomain)
}

func TestAccRRsetResource_invalidTypeLowercase(t *testing.T) {
	domainName := testAccDomainName(t, "rec-val-type")
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

resource "desec_rrset" "test" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "a"
  ttl     = 3600
  rdata = ["1.2.3.4"]
}
`, providerConfig, domainName),
				ExpectError: regexp.MustCompile(`must be an uppercase DNS record type`),
			},
		},
	})
}

func TestAccRRsetResource_invalidTTLZero(t *testing.T) {
	domainName := testAccDomainName(t, "rec-val-ttl")
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

resource "desec_rrset" "test" {
  domain  = desec_domain.test.name
  subname = "www"
  type    = "A"
  ttl     = 0
  rdata = ["1.2.3.4"]
}
`, providerConfig, domainName),
				ExpectError: regexp.MustCompile(`must be at least 1`),
			},
		},
	})
}
