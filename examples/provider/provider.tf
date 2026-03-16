provider "desec" {
  api_token = "your-desec-api-token"
  # api_url = "https://desec.io/api/v1"  # optional, defaults to the deSEC production API
}

# Create a domain and retrieve the nameservers assigned by deSEC
# to enter into your domain registrar settings.
resource "desec_domain" "example" {
  name = "example.com"
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

# Lazy provider initialization: use a token created by one provider instance
# to configure a second provider instance. The second provider's api_token is
# unknown at plan time, so it is configured lazily at apply time.
#
# The bootstrap provider creates the domain and all token scoping policies.
# Once the scoped token and its policies are in place, the lazily-initialized
# provider uses that token to manage DNS records for the domain.
provider "desec" {
  alias     = "bootstrap"
  api_token = "your-desec-api-token"
}

provider "desec" {
  api_token = desec_token.lazy_token.token
}

resource "desec_token" "lazy_token" {
  provider = desec.bootstrap
  name     = "lazy-init-token"
}

# Default policy: deny write access to all RRsets by default.
# This must be created before any specific policies.
resource "desec_token_policy" "lazy_token_default" {
  provider   = desec.bootstrap
  token_id   = desec_token.lazy_token.id
  perm_write = false
}

# Bootstrap creates the domain so the domain-specific policy can reference it.
resource "desec_domain" "example" {
  provider = desec.bootstrap
  name     = "test.example.dedyn.io"
}

# Allow the lazy token to write only the target domain.
resource "desec_token_policy" "lazy_token_domain" {
  provider   = desec.bootstrap
  token_id   = desec_token.lazy_token.id
  domain     = desec_domain.example.name
  perm_write = true

  depends_on = [desec_token_policy.lazy_token_default, desec_domain.example]
}

# The lazy provider manages records on the domain, proving it initialized correctly.
resource "desec_record" "example_a" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "A"
  records = ["203.0.113.1"]
  ttl     = 3600

  depends_on = [desec_token_policy.lazy_token_domain]
}

# Internationalized Domain Names (IDN): use to_punycode() to convert a unicode
# domain name to its Punycode (ACE) form before registering it with deSEC.
# The deSEC API only accepts domain names in Punycode form.
resource "desec_domain" "idn_example" {
  name = provider::desec::to_punycode("münchen.de")
}

# from_punycode() converts back to the human-readable unicode form for display.
output "idn_domain_unicode" {
  description = "The IDN domain name in human-readable unicode form."
  value       = provider::desec::from_punycode(desec_domain.idn_example.name)
}
