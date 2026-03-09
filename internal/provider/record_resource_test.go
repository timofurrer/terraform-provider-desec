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
