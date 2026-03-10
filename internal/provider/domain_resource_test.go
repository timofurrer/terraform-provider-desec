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

func TestAccDomainResource(t *testing.T) {
	domainName := testAccDomainName(t, "acc-test")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and Read testing.
			{
				Config: testAccDomainResourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("minimum_ttl"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("keys"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing.
			{
				ResourceName:      "desec_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
				// published and touched are volatile server-side timestamps that
				// can change asynchronously between Create and a subsequent Read
				// (e.g. deSEC publishes the initial SOA/NS records shortly after
				// domain creation). Ignore them during ImportStateVerify.
				ImportStateVerifyIgnore: []string{"published", "touched"},
			},
		},
	})
}

func TestAccDomainResourceIdentity(t *testing.T) {
	domainName := testAccDomainName(t, "id-acc-test")
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
				Config: testAccDomainResourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						"desec_domain.test",
						map[string]knownvalue.Check{
							"name": knownvalue.StringExact(domainName),
						},
					),
				},
			},
			// Import using identity.
			{
				ResourceName:    "desec_domain.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccDomainResourceConfig(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "test" {
  name = %q
}
`, providerConfig, name)
}

// TestAccDomainResource_OutOfBandDelete verifies that when a domain is deleted
// outside of Terraform (e.g. via the API directly), the provider detects the
// drift on the next plan and removes the resource from state.
func TestAccDomainResource_OutOfBandDelete(t *testing.T) {
	domainName := testAccDomainName(t, "oob-del")
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create the domain.
			{
				Config: testAccDomainResourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName),
					),
				},
			},
			// Delete the domain out-of-band, then apply the same config again.
			// The provider should detect that the domain is gone and recreate it.
			{
				PreConfig: func() {
					if err := client.DeleteDomain(context.Background(), domainName); err != nil {
						t.Fatalf("out-of-band domain delete: %v", err)
					}
				},
				Config: testAccDomainResourceConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName),
					),
				},
			},
		},
	})
}

// TestAccDomainResource_Recreate verifies that changing the domain name
// (an immutable field) triggers a destroy-and-recreate cycle.
func TestAccDomainResource_Recreate(t *testing.T) {
	domainName1 := testAccDomainName(t, "recreate-a")
	domainName2 := testAccDomainName(t, "recreate-b")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create with the first name.
			{
				Config: testAccDomainResourceConfig(providerConfig, domainName1),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName1),
					),
				},
			},
			// Change to a different name — this must trigger destroy+create.
			{
				Config: testAccDomainResourceConfig(providerConfig, domainName2),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(domainName2),
					),
					statecheck.ExpectKnownValue(
						"desec_domain.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(domainName2),
					),
				},
			},
		},
	})
}
