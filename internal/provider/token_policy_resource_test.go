// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccTokenPolicyResourceDefault(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and Read: default policy (no domain/subname/type).
			{
				Config: testAccTokenPolicyResourceDefaultConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("token_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("domain"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("subname"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("type"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("perm_write"),
						knownvalue.Bool(false),
					),
				},
			},
			// ImportState: format is "token_id/policy_id".
			{
				ResourceName:      "desec_token_policy.default",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccTokenPolicyImportID("desec_token_policy.default"),
			},
			// Update: flip perm_write to true.
			{
				Config: testAccTokenPolicyResourceDefaultPermWriteConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("perm_write"),
						knownvalue.Bool(true),
					),
					// domain/subname/type remain null after update.
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("domain"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

func TestAccTokenPolicyResourceSpecific(t *testing.T) {
	domainName := testAccDomainName(t, "policy-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create default policy first, then a specific policy for a domain.
			{
				Config: testAccTokenPolicyResourceSpecificConfig(providerConfig, domainName, true),
				ConfigStateChecks: []statecheck.StateCheck{
					// Default policy checks.
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("domain"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.default",
						tfjsonpath.New("perm_write"),
						knownvalue.Bool(false),
					),
					// Specific policy checks.
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("subname"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("type"),
						knownvalue.Null(),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("perm_write"),
						knownvalue.Bool(true),
					),
				},
			},
			// Update perm_write on the specific policy.
			{
				Config: testAccTokenPolicyResourceSpecificConfig(providerConfig, domainName, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("perm_write"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func TestAccTokenPolicyResourceSubnameAndType(t *testing.T) {
	domainName := testAccDomainName(t, "policy-st-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenPolicyResourceSubnameTypeConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("www"),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
					statecheck.ExpectKnownValue(
						"desec_token_policy.specific",
						tfjsonpath.New("perm_write"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

func TestAccTokenPolicyResourceIdentity(t *testing.T) {
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
				Config: testAccTokenPolicyResourceDefaultConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentityValue(
						"desec_token_policy.default",
						tfjsonpath.New("token_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectIdentityValue(
						"desec_token_policy.default",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Import using identity.
			{
				ResourceName:    "desec_token_policy.default",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// testAccTokenPolicyImportID returns an ImportStateIdFunc that builds the
// "token_id/policy_id" composite import ID from the resource's state.
func testAccTokenPolicyImportID(resourceAddr string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceAddr]
		if !ok {
			return "", fmt.Errorf("resource %q not found in state", resourceAddr)
		}
		tokenID := rs.Primary.Attributes["token_id"]
		policyID := rs.Primary.Attributes["id"]
		return tokenID + "/" + policyID, nil
	}
}

func testAccTokenPolicyResourceDefaultConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name = "policy-test-token"
}

resource "desec_token_policy" "default" {
  token_id = desec_token.test.id
}
`, providerConfig)
}

func testAccTokenPolicyResourceDefaultPermWriteConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name = "policy-test-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = true
}
`, providerConfig)
}

func testAccTokenPolicyResourceDefaultPermWriteFalseConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name = "policy-test-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = false
}
`, providerConfig)
}

func testAccTokenPolicyResourceSpecificConfig(providerConfig, domainName string, permWrite bool) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_token" "test" {
  name = "policy-test-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = false
}

resource "desec_token_policy" "specific" {
  token_id   = desec_token.test.id
  domain     = desec_domain.test.name
  perm_write = %t

  depends_on = [desec_token_policy.default]
}
`, providerConfig, domainName, permWrite)
}

func testAccTokenPolicyResourceSubnameTypeConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_token" "test" {
  name = "policy-st-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = false
}

resource "desec_token_policy" "specific" {
  token_id   = desec_token.test.id
  domain     = desec_domain.test.name
  subname    = "www"
  type       = "A"
  perm_write = true

  depends_on = [desec_token_policy.default]
}
`, providerConfig, domainName)
}

// TestAccTokenPolicyResource_UpdatePermWriteMultiStep verifies perm_write can
// be toggled on the default policy across multiple update steps.
func TestAccTokenPolicyResource_UpdatePermWriteMultiStep(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Step 1: create with perm_write = false.
			{
				Config: testAccTokenPolicyResourceDefaultConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("perm_write"), knownvalue.Bool(false)),
				},
			},
			// Step 2: update to perm_write = true.
			{
				Config: testAccTokenPolicyResourceDefaultPermWriteConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("perm_write"), knownvalue.Bool(true)),
					// Scope fields must still be null.
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("domain"), knownvalue.Null()),
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("subname"), knownvalue.Null()),
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("type"), knownvalue.Null()),
				},
			},
			// Step 3: update back to perm_write = false.
			{
				Config: testAccTokenPolicyResourceDefaultPermWriteFalseConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("perm_write"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

// TestAccTokenPolicyResource_SpecificImport verifies that a specific (non-default)
// policy can be imported using the "token_id/policy_id" composite ID.
func TestAccTokenPolicyResource_SpecificImport(t *testing.T) {
	domainName := testAccDomainName(t, "policy-imp-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenPolicyResourceSpecificConfig(providerConfig, domainName, true),
			},
			// Import the specific policy.
			{
				ResourceName:      "desec_token_policy.specific",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccTokenPolicyImportID("desec_token_policy.specific"),
			},
		},
	})
}

// TestAccTokenPolicyResource_MultipleSpecific verifies that one token can have
// a default policy and several specific domain policies simultaneously.
func TestAccTokenPolicyResource_MultipleSpecific(t *testing.T) {
	domain1 := testAccDomainName(t, "policy-multi1-acc")
	domain2 := testAccDomainName(t, "policy-multi2-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenPolicyResourceMultipleSpecificConfig(providerConfig, domain1, domain2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("domain"), knownvalue.Null()),
					statecheck.ExpectKnownValue("desec_token_policy.specific1", tfjsonpath.New("domain"), knownvalue.StringExact(domain1)),
					statecheck.ExpectKnownValue("desec_token_policy.specific1", tfjsonpath.New("perm_write"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token_policy.specific2", tfjsonpath.New("domain"), knownvalue.StringExact(domain2)),
					statecheck.ExpectKnownValue("desec_token_policy.specific2", tfjsonpath.New("perm_write"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

// TestAccTokenPolicyResource_DeleteSpecificBeforeDefault verifies that specific
// policies are destroyed before the default policy (Terraform dependency order).
func TestAccTokenPolicyResource_DeleteSpecificBeforeDefault(t *testing.T) {
	domainName := testAccDomainName(t, "policy-del-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create both policies.
			{
				Config: testAccTokenPolicyResourceSpecificConfig(providerConfig, domainName, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("domain"), knownvalue.Null()),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("domain"), knownvalue.StringExact(domainName)),
				},
			},
			// Remove the specific policy only, leaving the default.
			{
				Config: testAccTokenPolicyResourceDefaultConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("domain"), knownvalue.Null()),
				},
			},
		},
	})
}

// TestAccTokenPolicyResource_SpecificPolicySubnameOnly creates a policy
// scoped to a domain+subname with no type restriction.
func TestAccTokenPolicyResource_SpecificPolicySubnameOnly(t *testing.T) {
	domainName := testAccDomainName(t, "policy-sn-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenPolicyResourceSubnameOnlyConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("domain"), knownvalue.StringExact(domainName)),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("subname"), knownvalue.StringExact("mail")),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("type"), knownvalue.Null()),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("perm_write"), knownvalue.Bool(true)),
				},
			},
		},
	})
}

// TestAccTokenPolicyResource_SpecificPolicyTypeOnly creates a policy scoped
// to a domain+type with no subname restriction.
func TestAccTokenPolicyResource_SpecificPolicyTypeOnly(t *testing.T) {
	domainName := testAccDomainName(t, "policy-ty-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenPolicyResourceTypeOnlyConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("domain"), knownvalue.StringExact(domainName)),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("subname"), knownvalue.Null()),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("type"), knownvalue.StringExact("AAAA")),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("perm_write"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

// TestAccTokenPolicyResource_OutOfBandDelete verifies that when a token policy
// is deleted outside of Terraform, the provider detects the drift and recreates it.
func TestAccTokenPolicyResource_OutOfBandDelete(t *testing.T) {
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	var capturedTokenID, capturedPolicyID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create the default policy.
			{
				Config: testAccTokenPolicyResourceDefaultConfig(providerConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("desec_token_policy.default", "token_id", func(v string) error {
						capturedTokenID = v
						return nil
					}),
					resource.TestCheckResourceAttrWith("desec_token_policy.default", "id", func(v string) error {
						capturedPolicyID = v
						return nil
					}),
				),
			},
			// Delete out-of-band, then re-apply. Provider should recreate it.
			{
				PreConfig: func() {
					if capturedTokenID == "" || capturedPolicyID == "" {
						t.Fatal("capturedTokenID or capturedPolicyID was not set")
					}
					if err := client.DeleteTokenPolicy(context.Background(), capturedTokenID, capturedPolicyID); err != nil {
						t.Fatalf("out-of-band token policy delete: %v", err)
					}
				},
				Config: testAccTokenPolicyResourceDefaultConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("perm_write"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token_policy.default", tfjsonpath.New("domain"), knownvalue.Null()),
				},
			},
		},
	})
}

// TestAccTokenPolicyResource_PolicyScopeReplace verifies that changing an
// immutable scope field (domain) triggers a destroy+recreate of the policy.
func TestAccTokenPolicyResource_PolicyScopeReplace(t *testing.T) {
	domain1 := testAccDomainName(t, "policy-rep1-acc")
	domain2 := testAccDomainName(t, "policy-rep2-acc")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create specific policy for domain1.
			{
				Config: testAccTokenPolicyResourceSingleSpecificConfig(providerConfig, domain1, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("domain"), knownvalue.StringExact(domain1)),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			// Change to domain2 — must destroy and recreate.
			{
				Config: testAccTokenPolicyResourceSingleSpecificConfig(providerConfig, domain2, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("domain"), knownvalue.StringExact(domain2)),
					statecheck.ExpectKnownValue("desec_token_policy.specific", tfjsonpath.New("perm_write"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

// ---- additional config helpers ----

func testAccTokenPolicyResourceMultipleSpecificConfig(providerConfig, domain1, domain2 string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "domain1" {
  name = %q
}

resource "desec_domain" "domain2" {
  name = %q
}

resource "desec_token" "test" {
  name = "multi-policy-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = false
}

resource "desec_token_policy" "specific1" {
  token_id   = desec_token.test.id
  domain     = desec_domain.domain1.name
  perm_write = true

  depends_on = [desec_token_policy.default]
}

resource "desec_token_policy" "specific2" {
  token_id   = desec_token.test.id
  domain     = desec_domain.domain2.name
  perm_write = false

  depends_on = [desec_token_policy.default]
}
`, providerConfig, domain1, domain2)
}

func testAccTokenPolicyResourceSubnameOnlyConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_token" "test" {
  name = "subname-policy-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = false
}

resource "desec_token_policy" "specific" {
  token_id   = desec_token.test.id
  domain     = desec_domain.test.name
  subname    = "mail"
  perm_write = true

  depends_on = [desec_token_policy.default]
}
`, providerConfig, domainName)
}

func testAccTokenPolicyResourceTypeOnlyConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}

resource "desec_token" "test" {
  name = "type-policy-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = false
}

resource "desec_token_policy" "specific" {
  token_id   = desec_token.test.id
  domain     = desec_domain.test.name
  type       = "AAAA"
  perm_write = false

  depends_on = [desec_token_policy.default]
}
`, providerConfig, domainName)
}

// testAccTokenPolicyResourceSingleSpecificConfig creates a default + one specific
// policy (used for the scope-replace test). Two domains are always declared so
// both domain names are always valid resources, even when only one is used by
// the specific policy at each step.
func testAccTokenPolicyResourceSingleSpecificConfig(providerConfig, specificDomain string, permWrite bool) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "specific" {
  name = %q
}

resource "desec_token" "test" {
  name = "scope-replace-token"
}

resource "desec_token_policy" "default" {
  token_id   = desec_token.test.id
  perm_write = false
}

resource "desec_token_policy" "specific" {
  token_id   = desec_token.test.id
  domain     = desec_domain.specific.name
  perm_write = %t

  depends_on = [desec_token_policy.default]
}
`, providerConfig, specificDomain, permWrite)
}
