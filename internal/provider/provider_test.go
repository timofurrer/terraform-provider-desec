// Copyright Timo Furrer 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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

func TestAccProviderExample_DomainNameservers(t *testing.T) {
	domainName := testAccDomainName(t, "ns-example")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderExampleDomainNameserversConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.desec_record.nameservers",
						tfjsonpath.New("domain"),
						knownvalue.StringExact(domainName),
					),
					statecheck.ExpectKnownValue(
						"data.desec_record.nameservers",
						tfjsonpath.New("subname"),
						knownvalue.StringExact("@"),
					),
					statecheck.ExpectKnownValue(
						"data.desec_record.nameservers",
						tfjsonpath.New("type"),
						knownvalue.StringExact("NS"),
					),
					statecheck.ExpectKnownValue(
						"data.desec_record.nameservers",
						tfjsonpath.New("records"),
						knownvalue.SetSizeExact(2),
					),
					statecheck.ExpectKnownOutputValue(
						"nameservers",
						knownvalue.SetSizeExact(2),
					),
				},
			},
		},
	})
}

func TestAccProviderExample_DNSSEC(t *testing.T) {
	domainName := testAccDomainName(t, "dnssec-example")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderExampleDNSSECConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					// DS records are a flat list of strings with the format:
					// "<keytag> <algorithm> <digest_type> <hex_digest>"
					statecheck.ExpectKnownOutputValue(
						"dnssec_ds_records",
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.StringRegexp(regexp.MustCompile(`^\d+ \d+ \d+ [0-9a-fA-F]+$`)),
						}),
					),
					// DNSKEY records have the format:
					// "<flags> <protocol> <algorithm> <base64_public_key>"
					statecheck.ExpectKnownOutputValue(
						"dnssec_dnskeys",
						knownvalue.ListPartial(map[int]knownvalue.Check{
							0: knownvalue.StringRegexp(regexp.MustCompile(`^\d+ \d+ \d+ [A-Za-z0-9+/]+=*$`)),
						}),
					),
				},
			},
		},
	})
}

func TestAccProviderExample_BulkRecords(t *testing.T) {
	domainName := testAccDomainName(t, "bulk-records-example")
	providerConfig, factories := newTestAccEnv(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderExampleBulkRecordsConfig(providerConfig, domainName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"desec_records.bulk_example",
						tfjsonpath.New("records"),
						knownvalue.SetSizeExact(3),
					),
					statecheck.ExpectKnownValue(
						"desec_records.bulk_example",
						tfjsonpath.New("exclusive"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownOutputValue(
						"managed_zonefile",
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccProviderExampleBulkRecordsConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "bulk_example" {
  name = %q
}

resource "desec_records" "bulk_example" {
  domain    = desec_domain.bulk_example.name
  exclusive = true

  records = [
    {
      subname = ""
      type    = "A"
      ttl     = 3600
      records = ["203.0.113.10"]
    },
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      records = ["203.0.113.10"]
    },
    {
      subname = ""
      type    = "MX"
      ttl     = 3600
      records = ["10 mail.example.com."]
    },
  ]
}

output "managed_zonefile" {
  description = "The canonical zone file computed from the structured records."
  value       = desec_records.bulk_example.zonefile
}
`, providerConfig, domainName)
}

func testAccProviderExampleDNSSECConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "example" {
  name = %q
}

output "dnssec_ds_records" {
  description = "DS records for DNSSEC delegation — enter these at your domain registrar."
  value       = flatten([for key in desec_domain.example.keys : key.ds if key.managed])
}

output "dnssec_dnskeys" {
  description = "DNSKEY public key records for the domain."
  value       = [for key in desec_domain.example.keys : key.dnskey if key.managed]
}
`, providerConfig, domainName)
}

func testAccProviderExampleDomainNameserversConfig(providerConfig, domainName string) string {
	return fmt.Sprintf(`
%s

resource "desec_domain" "example" {
  name = %q
}

data "desec_record" "nameservers" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "NS"
}

output "nameservers" {
  description = "The deSEC nameservers to enter at your domain registrar."
  value       = data.desec_record.nameservers.records
}
`, providerConfig, domainName)
}
