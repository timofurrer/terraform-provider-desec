// Copyright (c) Timo Furrer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/timofurrer/terraform-provider-desec/internal/api"
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
	config, factories, _ := newTestAccEnvWithClient(t)
	return config, factories
}

// newTestAccEnvWithClient is like newTestAccEnv but also returns an *api.Client
// configured against the same backend (fake or real). This allows tests to make
// direct API calls for out-of-band operations such as deleting a resource to
// test drift detection.
func newTestAccEnvWithClient(t *testing.T) (string, map[string]func() (tfprotov6.ProviderServer, error), *api.Client) {
	t.Helper()

	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"desec": providerserver.NewProtocol6WithError(New("test")()),
	}

	if useRealAPI() {
		// Real API: token and URL come from environment variables.
		apiToken := os.Getenv("DESEC_API_TOKEN")
		apiURL := os.Getenv("DESEC_API_URL")
		if apiURL == "" {
			apiURL = api.DefaultBaseURL
		}
		client := api.NewClient(apiURL, apiToken)
		return `provider "desec" {}`, factories, client
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

	client := api.NewClient(srv.URL(), srv.Token())
	return config, factories, client
}
