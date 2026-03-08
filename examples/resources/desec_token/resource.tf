# Create a token for CI/CD deployments with domain management permissions
resource "desec_token" "ci" {
  name               = "ci-deploy"
  perm_create_domain = true
  perm_delete_domain = true
}

# The secret value is only available right after creation.
# Store it securely (e.g. in a secrets manager).
output "ci_token_secret" {
  value     = desec_token.ci.token
  sensitive = true
}

# Create a token restricted to a specific source IP subnet
resource "desec_token" "restricted" {
  name            = "office-only"
  allowed_subnets = ["203.0.113.0/24"]
}

# Create a token that expires after 90 days of inactivity
resource "desec_token" "expiring" {
  name              = "short-lived"
  max_unused_period = "90 00:00:00"
}
