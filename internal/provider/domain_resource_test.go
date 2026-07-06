// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
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

// TestAccDomainResource_Punycode verifies that a domain name in Punycode /
// ACE form (e.g. "xn--mnchen-3ya" — the Punycode encoding of "münchen") is
// accepted and round-trips through the API unchanged.
//
// "xn--mnchen-3ya" is the Punycode encoding of the German word "münchen".
// Using it as a label in a dedyn.io subdomain gives a fully valid DNS name
// that can be registered against the real deSEC API.
func TestAccDomainResource_Punycode(t *testing.T) {
	// testAccDomainName with a punycode suffix produces:
	//   fake mode : xn--mnchen-3ya.example.com
	//   real API  : tf-acc-xn--mnchen-3ya-<test>-<random>.dedyn.io
	domainName := testAccDomainName(t, "xn--mnchen-3ya")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and Read: the Punycode name must round-trip unchanged.
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
						tfjsonpath.New("keys"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState: the imported name must also be the Punycode form.
			{
				ResourceName:            "desec_domain.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"published", "touched"},
			},
		},
	})
}

// TestAccDomainResource_UnicodeRejected verifies that supplying a domain name
// with non-ASCII (unicode/umlaut) characters is rejected at plan time with a
// clear, actionable error message directing the user to use Punycode instead.
//
// deSEC only accepts IDN domains in Punycode form; the provider enforces this
// up-front so users get a helpful error rather than an opaque API 400.
//
// testAccDomainName with a unicode suffix produces a name that contains a
// non-ASCII character in both fake and real-API mode:
//
//	fake mode : münchen.example.com
//	real API  : tf-acc-münchen-<test>-<random>.dedyn.io
func TestAccDomainResource_UnicodeRejected(t *testing.T) {
	// Use testAccDomainName so the name follows the same pattern as all other
	// tests (example.com in fake mode, dedyn.io in real-API mode). The unicode
	// suffix ensures the name contains a non-ASCII character in both modes.
	domainName := testAccDomainName(t, "münchen")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccDomainResourceConfig(providerConfig, domainName),
				// The provider must reject this at plan time with the exact
				// error title, explanation, and actionable to_punycode()
				// suggestion. (?s) makes . match newlines so the full
				// multi-line diagnostic body is covered. Terraform may
				// word-wrap long lines, so each distinct sentence or clause
				// is matched separately with .* in between.
				ExpectError: regexp.MustCompile(
					`(?s)Non-ASCII characters in domain name` +
						`.*The domain name` +
						`.*` + regexp.QuoteMeta(domainName) +
						`.*contains` +
						`.*non-ASCII characters` +
						`.*only accepts domain names in Punycode \(ACE\) form\.` +
						`.*Use the provider::desec::to_punycode\(\) function to convert it automatically:` +
						`.*provider::desec::to_punycode\(` +
						`.*` + regexp.QuoteMeta(domainName) +
						`.*\)`,
				),
			},
		},
	})
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

// TestDomainResourceRead_NotFoundSetsIdentity calls Read directly, with a
// null starting identity, for a domain that has been deleted out-of-band,
// and verifies that the not-found branch (which removes the resource from
// state) also sets identity. Without this, a subsequent plan fails with
// "Missing Resource Identity After Read" instead of proposing recreation.
// Calling through terraform-plugin-testing wouldn't exercise this: the
// framework pre-populates a response's identity from the prior identity
// before invoking the resource, so once Create had set identity correctly,
// that carry-over would mask Read's not-found branch not setting it.
func TestDomainResourceRead_NotFoundSetsIdentity(t *testing.T) {
	ctx := context.Background()
	client := newIdentityTestClient(t)

	const domainName = "read-notfound-domain-identity-example.dedyn.io"
	domain, err := client.CreateDomain(ctx, api.CreateDomainOptions{Name: domainName})
	if err != nil {
		t.Fatalf("create domain: %v", err)
	}

	var prior domainResourceModel
	if diags := domainToModel(domain, &prior); diags.HasError() {
		t.Fatalf("build prior model: %v", diags)
	}

	if err := client.DeleteDomain(ctx, domainName); err != nil {
		t.Fatalf("delete domain out of band: %v", err)
	}

	res := newDomainResource()
	configureTestResource(t, ctx, res, client)
	schema := resourceSchema(ctx, res)

	state := tfsdk.State{Schema: schema.Schema}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("build prior state: %v", diags)
	}

	readReq := fwresource.ReadRequest{
		State:    state,
		Identity: nullResourceIdentity(t, ctx, res),
	}
	readResp := fwresource.ReadResponse{
		State:    tfsdk.State{Schema: schema.Schema},
		Identity: nullResourceIdentity(t, ctx, res),
	}

	res.Read(ctx, readReq, &readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read returned diagnostics: %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatal("expected Read to remove the resource from state after out-of-band deletion")
	}
	requireNonNullIdentity(t, readResp.Identity)

	var gotIdentity domainIdentityModel
	if diags := readResp.Identity.Get(ctx, &gotIdentity); diags.HasError() {
		t.Fatalf("read back identity: %v", diags)
	}
	if gotIdentity.Name.ValueString() != domainName {
		t.Fatalf("unexpected identity: %+v", gotIdentity)
	}
}
