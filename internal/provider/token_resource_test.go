// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"sync"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccTokenResource(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and Read testing.
			{
				Config: testAccTokenResourceConfig(providerConfig, "my-token", true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("my-token"),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("token"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("created"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("owner"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("is_valid"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_create_domain"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_delete_domain"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_manage_tokens"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing — token secret cannot be recovered, so ignore it.
			{
				ResourceName:            "desec_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update: change name and permissions.
			{
				Config: testAccTokenResourceConfig(providerConfig, "updated-token", false, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("updated-token"),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_create_domain"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("perm_delete_domain"),
						knownvalue.Bool(true),
					),
					// Token secret must still be present in state after update.
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("token"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccTokenResourceWithSubnets(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenResourceWithSubnetsConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("192.168.0.0/24"),
						}),
					),
				},
			},
		},
	})
}

func TestAccTokenResourceIdentity(t *testing.T) {
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
				Config: testAccTokenResourceConfig(providerConfig, "identity-token", true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentityValue(
						"desec_token.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Import using identity.
			{
				ResourceName:    "desec_token.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccTokenResourceConfig(providerConfig, name string, permCreate, permDelete bool) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name               = %q
  perm_create_domain = %t
  perm_delete_domain = %t
}
`, providerConfig, name, permCreate, permDelete)
}

func testAccTokenResourceWithSubnetsConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name            = "subnet-token"
  allowed_subnets = ["192.168.0.0/24"]
}
`, providerConfig)
}

// TestAccTokenResource_UpdatePerms cycles through every permission flag
// independently to ensure each can be toggled without affecting others.
func TestAccTokenResource_UpdatePerms(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create with all perms false.
			{
				Config: testAccTokenResourceAllPermsConfig(providerConfig, false, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_create_domain"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_delete_domain"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_manage_tokens"), knownvalue.Bool(false)),
				},
			},
			// Enable perm_create_domain only.
			{
				Config: testAccTokenResourceAllPermsConfig(providerConfig, true, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_create_domain"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_delete_domain"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_manage_tokens"), knownvalue.Bool(false)),
				},
			},
			// Enable perm_delete_domain only.
			{
				Config: testAccTokenResourceAllPermsConfig(providerConfig, false, true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_create_domain"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_delete_domain"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_manage_tokens"), knownvalue.Bool(false)),
				},
			},
			// Enable perm_manage_tokens only.
			{
				Config: testAccTokenResourceAllPermsConfig(providerConfig, false, false, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_create_domain"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_delete_domain"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_manage_tokens"), knownvalue.Bool(true)),
				},
			},
			// Enable all perms.
			{
				Config: testAccTokenResourceAllPermsConfig(providerConfig, true, true, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_create_domain"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_delete_domain"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_manage_tokens"), knownvalue.Bool(true)),
				},
			},
			// Disable all perms.
			{
				Config: testAccTokenResourceAllPermsConfig(providerConfig, false, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_create_domain"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_delete_domain"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_manage_tokens"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

// TestAccTokenResource_UpdateAutoPolicy verifies that the auto_policy flag can
// be toggled via update.
func TestAccTokenResource_UpdateAutoPolicy(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenResourceAutoPolicyConfig(providerConfig, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("auto_policy"), knownvalue.Bool(false)),
				},
			},
			{
				Config: testAccTokenResourceAutoPolicyConfig(providerConfig, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("auto_policy"), knownvalue.Bool(true)),
				},
			},
			{
				Config: testAccTokenResourceAutoPolicyConfig(providerConfig, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("auto_policy"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

// TestAccTokenResource_UpdateAllowedSubnets verifies that the allowed_subnets
// list can be added to, extended, and removed across multiple update steps.
func TestAccTokenResource_UpdateAllowedSubnets(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create with a single subnet.
			{
				Config: testAccTokenResourceSubnetsConfig(providerConfig, `"10.0.0.0/8"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/8"),
						}),
					),
				},
			},
			// Add a second subnet.
			{
				Config: testAccTokenResourceSubnetsConfig(providerConfig, `"10.0.0.0/8", "192.168.0.0/16"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.ListSizeExact(2),
					),
				},
			},
			// Reduce back to one different subnet.
			{
				Config: testAccTokenResourceSubnetsConfig(providerConfig, `"172.16.0.0/12"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("172.16.0.0/12"),
						}),
					),
				},
			},
		},
	})
}

// TestAccTokenResource_MaxAge verifies that the max_age attribute can be set,
// updated to a different value, and cleared (set to null).
func TestAccTokenResource_MaxAge(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create with max_age set.
			{
				Config: testAccTokenResourceMaxAgeConfig(providerConfig, `"365 00:00:00"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("max_age"),
						knownvalue.StringExact("365 00:00:00"),
					),
				},
			},
			// Change to a different value.
			{
				Config: testAccTokenResourceMaxAgeConfig(providerConfig, `"30 00:00:00"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("max_age"),
						knownvalue.StringExact("30 00:00:00"),
					),
				},
			},
			// Remove (set to null).
			{
				Config: testAccTokenResourceNoMaxAgeConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("max_age"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

// TestAccTokenResource_MaxUnusedPeriod verifies that the max_unused_period
// attribute can be set, updated, and cleared.
func TestAccTokenResource_MaxUnusedPeriod(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenResourceMaxUnusedPeriodConfig(providerConfig, `"90 00:00:00"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("max_unused_period"),
						knownvalue.StringExact("90 00:00:00"),
					),
				},
			},
			{
				Config: testAccTokenResourceMaxUnusedPeriodConfig(providerConfig, `"14 00:00:00"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("max_unused_period"),
						knownvalue.StringExact("14 00:00:00"),
					),
				},
			},
			// Remove (set to null).
			{
				Config: testAccTokenResourceNoMaxUnusedPeriodConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("max_unused_period"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

// TestAccTokenResource_UpdateName verifies that renaming a token updates the
// name attribute without replacing the token.
func TestAccTokenResource_UpdateName(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenResourceConfig(providerConfig, "original-name", false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("name"), knownvalue.StringExact("original-name")),
				},
			},
			{
				Config: testAccTokenResourceConfig(providerConfig, "renamed-token", false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("name"), knownvalue.StringExact("renamed-token")),
				},
			},
			{
				Config: testAccTokenResourceConfig(providerConfig, "renamed-again", false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("name"), knownvalue.StringExact("renamed-again")),
				},
			},
		},
	})
}

// TestAccTokenResource_TokenSecretPreserved verifies that the token secret
// stays identical in state across multiple update operations. This is the key
// guarantee that rotation does not happen on updates.
func TestAccTokenResource_TokenSecretPreserved(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	// captureSecret captures the token secret after step 1 so that subsequent
	// steps can verify it hasn't changed.
	capture := &captureTokenSecret{addr: "desec_token.test"}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create and capture the secret.
			{
				Config: testAccTokenResourceAllPermsConfig(providerConfig, false, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					capture,
				},
			},
			// Update name — secret must be unchanged.
			{
				Config: testAccTokenResourceConfig(providerConfig, "renamed", false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					capture.mustEqual(),
				},
			},
			// Update perms — secret must still be unchanged.
			{
				Config: testAccTokenResourceAllPermsConfig(providerConfig, true, true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					capture.mustEqual(),
				},
			},
			// Update subnets — secret must still be unchanged.
			{
				Config: testAccTokenResourceSubnetsConfig(providerConfig, `"10.0.0.0/8"`),
				ConfigStateChecks: []statecheck.StateCheck{
					capture.mustEqual(),
				},
			},
		},
	})
}

// TestAccTokenResource_OutOfBandDelete verifies that when a token is deleted
// outside of Terraform, the provider detects the drift and recreates it.
func TestAccTokenResource_OutOfBandDelete(t *testing.T) {
	providerConfig, factories, client := newTestAccEnvWithClient(t)

	// We need to track the token ID so we can delete it out-of-band.
	var capturedID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			// Create token.
			{
				Config: testAccTokenResourceConfig(providerConfig, "oob-del-token", false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("id"), knownvalue.NotNull()),
				},
				// Capture the ID after step completes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("desec_token.test", "id", func(val string) error {
						capturedID = val
						return nil
					}),
				),
			},
			// Delete out-of-band, then re-apply. Provider should recreate it.
			{
				PreConfig: func() {
					if capturedID == "" {
						t.Fatal("capturedID was not set")
					}
					if err := client.DeleteToken(context.Background(), capturedID); err != nil {
						t.Fatalf("out-of-band token delete: %v", err)
					}
				},
				Config: testAccTokenResourceConfig(providerConfig, "oob-del-token", false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("name"), knownvalue.StringExact("oob-del-token")),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("token"), knownvalue.NotNull()),
				},
			},
		},
	})
}

// TestAccTokenResource_AllAttributes creates a token with every optional
// attribute set and verifies that each one is reflected in state.
func TestAccTokenResource_AllAttributes(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenResourceFullConfig(providerConfig),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("name"), knownvalue.StringExact("full-token")),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_create_domain"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_delete_domain"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("perm_manage_tokens"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(
						"desec_token.test",
						tfjsonpath.New("allowed_subnets"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("10.0.0.0/8"),
							knownvalue.StringExact("192.168.0.0/16"),
						}),
					),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("auto_policy"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("max_age"), knownvalue.StringExact("365 00:00:00")),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("max_unused_period"), knownvalue.StringExact("30 00:00:00")),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("token"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("is_valid"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue("desec_token.test", tfjsonpath.New("owner"), knownvalue.NotNull()),
				},
			},
		},
	})
}

// ---- config helpers ----

func testAccTokenResourceAllPermsConfig(providerConfig string, permCreate, permDelete, permManage bool) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name               = "all-perms-token"
  perm_create_domain = %t
  perm_delete_domain = %t
  perm_manage_tokens = %t
}
`, providerConfig, permCreate, permDelete, permManage)
}

func testAccTokenResourceAutoPolicyConfig(providerConfig string, autoPolicy bool) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name        = "auto-policy-token"
  auto_policy = %t
}
`, providerConfig, autoPolicy)
}

func testAccTokenResourceSubnetsConfig(providerConfig, subnets string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name            = "subnet-update-token"
  allowed_subnets = [%s]
}
`, providerConfig, subnets)
}

func testAccTokenResourceMaxAgeConfig(providerConfig, maxAge string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name    = "max-age-token"
  max_age = %s
}
`, providerConfig, maxAge)
}

func testAccTokenResourceNoMaxAgeConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name = "max-age-token"
}
`, providerConfig)
}

func testAccTokenResourceMaxUnusedPeriodConfig(providerConfig, maxUnusedPeriod string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name              = "max-unused-token"
  max_unused_period = %s
}
`, providerConfig, maxUnusedPeriod)
}

func testAccTokenResourceNoMaxUnusedPeriodConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name = "max-unused-token"
}
`, providerConfig)
}

func testAccTokenResourceFullConfig(providerConfig string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "test" {
  name               = "full-token"
  perm_create_domain = true
  perm_delete_domain = true
  perm_manage_tokens = true
  allowed_subnets    = ["10.0.0.0/8", "192.168.0.0/16"]
  auto_policy        = true
  max_age            = "365 00:00:00"
  max_unused_period  = "30 00:00:00"
}
`, providerConfig)
}

// ---- statecheck helpers ----

// captureTokenSecret is a statecheck.StateCheck that, on first call, captures
// the token secret from state. Subsequent calls via mustEqual() verify it is
// unchanged.
type captureTokenSecret struct {
	addr   string
	mu     sync.Mutex
	secret string
}

func (c *captureTokenSecret) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	secret, err := extractTokenSecret(req.State, c.addr)
	if err != nil {
		resp.Error = err
		return
	}
	if c.secret == "" {
		c.secret = secret
	}
}

// mustEqual returns a StateCheck that verifies the current token secret equals
// the one captured earlier.
func (c *captureTokenSecret) mustEqual() statecheck.StateCheck {
	return &tokenSecretEqualsCheck{capture: c}
}

type tokenSecretEqualsCheck struct {
	capture *captureTokenSecret
}

func (c *tokenSecretEqualsCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	c.capture.mu.Lock()
	expected := c.capture.secret
	c.capture.mu.Unlock()

	current, err := extractTokenSecret(req.State, c.capture.addr)
	if err != nil {
		resp.Error = err
		return
	}
	if current != expected {
		resp.Error = fmt.Errorf("token secret changed: was %q, now %q", expected, current)
	}
}

func extractTokenSecret(state *tfjson.State, addr string) (string, error) {
	if state == nil || state.Values == nil || state.Values.RootModule == nil {
		return "", fmt.Errorf("state has no root module")
	}
	for _, r := range state.Values.RootModule.Resources {
		if r.Address == addr {
			v, ok := r.AttributeValues["token"]
			if !ok {
				return "", fmt.Errorf("resource %q has no attribute \"token\"", addr)
			}
			s, ok := v.(string)
			if !ok {
				return "", fmt.Errorf("resource %q attribute \"token\" is not a string (got %T)", addr, v)
			}
			return s, nil
		}
	}
	return "", fmt.Errorf("resource %q not found in state", addr)
}
