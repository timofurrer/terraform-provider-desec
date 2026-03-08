// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
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
