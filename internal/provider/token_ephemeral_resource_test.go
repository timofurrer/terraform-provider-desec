// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
)

func TestAccTokenEphemeralResource(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)
	factories["echo"] = echoprovider.NewProviderServer()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenEphemeralResourceConfig(providerConfig, "ephemeral-token"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("name"),
						knownvalue.StringExact("ephemeral-token"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("token"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("is_valid"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

func TestAccTokenEphemeralResourceKeepOnClose(t *testing.T) {
	providerConfig, factories := newTestAccEnv(t)
	factories["echo"] = echoprovider.NewProviderServer()

	// When running against the real API, the ephemeral resource intentionally
	// leaves the token alive (keep_on_close = true). Clean it up after the test
	// so it does not accumulate on the account across runs.
	if useRealAPI() {
		t.Cleanup(func() {
			apiToken := os.Getenv("DESEC_API_TOKEN")
			apiURL := os.Getenv("DESEC_API_URL")
			if apiURL == "" {
				apiURL = api.DefaultBaseURL
			}
			c := api.NewClient(apiURL, apiToken)
			tokens, err := c.ListTokens(context.Background())
			if err != nil {
				t.Logf("kept-token cleanup: ListTokens error: %v", err)
				return
			}
			for _, tok := range tokens {
				if tok.Name == "kept-token" {
					if err := c.DeleteToken(context.Background(), tok.ID); err != nil {
						t.Logf("kept-token cleanup: DeleteToken(%s) error: %v", tok.ID, err)
					}
				}
			}
		})
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccTokenEphemeralResourceKeepOnCloseConfig(providerConfig, "kept-token"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("token"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("keep_on_close"),
						knownvalue.Bool(true),
					),
				},
			},
		},
	})
}

func testAccTokenEphemeralResourceConfig(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

ephemeral "desec_token" "test" {
  name = %q
}

provider "echo" {
  data = ephemeral.desec_token.test
}

resource "echo" "test" {}
`, providerConfig, name)
}

func testAccTokenEphemeralResourceKeepOnCloseConfig(providerConfig, name string) string {
	return fmt.Sprintf(`
%s

ephemeral "desec_token" "test" {
  name         = %q
  keep_on_close = true
}

provider "echo" {
  data = ephemeral.desec_token.test
}

resource "echo" "test" {}
`, providerConfig, name)
}
