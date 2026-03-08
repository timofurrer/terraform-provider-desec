resource "desec_token" "ci" {
  name = "ci-deploy"
}

# Default policy: deny write access to all RRsets by default.
# This must be created before any specific policies.
resource "desec_token_policy" "default" {
  token_id   = desec_token.ci.id
  perm_write = false
}

# Allow the token to write all RRsets within example.com.
resource "desec_token_policy" "example_all" {
  token_id   = desec_token.ci.id
  domain     = "example.com"
  perm_write = true

  depends_on = [desec_token_policy.default]
}

# Allow the token to write only the www A record within example.com.
resource "desec_token_policy" "example_www_a" {
  token_id   = desec_token.ci.id
  domain     = "example.com"
  subname    = "www"
  type       = "A"
  perm_write = true

  depends_on = [desec_token_policy.default]
}
