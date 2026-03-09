provider "desec" {
  api_token = "your-desec-api-token"
  # api_url = "https://desec.io/api/v1"  # optional, defaults to the deSEC production API
}

# Lazy provider initialization: use a token created by one provider instance
# to configure a second provider instance. The second provider's api_token is
# unknown at plan time, so it is configured lazily at apply time.
# A token policy scoped to the exact target domain is created first, and the
# lazily-initialized provider depends on that policy being in place.
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

# Allow the token to write only test.example.com.
resource "desec_token_policy" "lazy_token_domain" {
  provider   = desec.bootstrap
  token_id   = desec_token.lazy_token.id
  domain     = "test.example.com"
  perm_write = true

  depends_on = [desec_token_policy.lazy_token_default]
}

resource "desec_domain" "example" {
  name = "test.example.com"

  # Ensure the scoping policy is in place before using the lazy provider.
  depends_on = [desec_token_policy.lazy_token_domain]
}
