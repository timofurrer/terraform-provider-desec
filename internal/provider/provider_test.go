// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/timofurrer/terraform-provider-desec/internal/api/fake"
)

// useRealAPI returns true when the DESEC_REAL_API environment variable is set.
func useRealAPI() bool {
	return os.Getenv("DESEC_REAL_API") != ""
}

// testAccPreCheck validates that the required environment variables are present
// when running against the real deSEC API.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if useRealAPI() {
		if os.Getenv("DESEC_API_TOKEN") == "" {
			t.Fatal("DESEC_API_TOKEN must be set when DESEC_REAL_API=1")
		}
	}
}

// newTestAccEnv returns a provider config HCL string and a provider factory map
// for use in a single acceptance test. When using the fake backend (the default),
// a fresh isolated fake server is started and automatically closed via t.Cleanup,
// so each test gets its own blank-slate server with no shared state.
func newTestAccEnv(t *testing.T) (string, map[string]func() (tfprotov6.ProviderServer, error)) {
	t.Helper()

	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"desec": providerserver.NewProtocol6WithError(New("test")()),
	}

	if useRealAPI() {
		// Real API: token and URL come from environment variables.
		return `provider "desec" {}`, factories
	}

	// Fake backend: start a fresh server for this test only.
	srv := fake.NewServer()
	t.Cleanup(srv.Close)

	config := fmt.Sprintf(`
provider "desec" {
  api_token = %q
  api_url   = %q
}
`, srv.Token(), srv.URL())

	return config, factories
}

// TestAccProviderLazyInit verifies that a provider instance can be lazily
// initialized with a token created by a separate, aliased provider instance.
// The default provider's api_token is sourced from an ephemeral desec_token
// resource opened by the bootstrap provider, so its value is unknown at plan
// time and the provider is configured only at apply time.
func TestAccProviderLazyInit(t *testing.T) {
	domainName := "test.example.com"

	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"desec": providerserver.NewProtocol6WithError(New("test")()),
	}

	var config string
	if useRealAPI() {
		config = `
provider "desec" {
  alias = "bootstrap"
}

provider "desec" {
  api_token = desec_token.lazy_token.token
}
`
	} else {
		srv := fake.NewServer()
		t.Cleanup(srv.Close)
		config = fmt.Sprintf(`
provider "desec" {
  alias     = "bootstrap"
  api_token = %q
  api_url   = %q
}

provider "desec" {
  api_token = desec_token.lazy_token.token
  api_url   = %q
}
`, srv.Token(), srv.URL(), srv.URL())
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderLazyInitConfig(config, domainName),
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

func testAccProviderLazyInitConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_token" "lazy_token" {
  provider = desec.bootstrap
  name     = "lazy-init-token"
}

# Default policy: deny write access to all RRsets by default.
resource "desec_token_policy" "lazy_token_default" {
  provider   = desec.bootstrap
  token_id   = desec_token.lazy_token.id
  perm_write = false
}

# Allow the token to write only the target domain.
resource "desec_token_policy" "lazy_token_domain" {
  provider   = desec.bootstrap
  token_id   = desec_token.lazy_token.id
  domain     = %q
  perm_write = true

  depends_on = [desec_token_policy.lazy_token_default]
}

resource "desec_domain" "test" {
  name = %q

  # Ensure the scoping policy is in place before using the lazy provider.
  depends_on = [desec_token_policy.lazy_token_domain]
}
`, providerConfig, domainName, domainName)
}
