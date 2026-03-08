# List all scoping policies for a token
data "desec_token_policies" "all" {
  token_id = "3a6b94b5-d20e-40bd-a7cc-521f5c79fab3"
}

# Output all domains that have an explicit write-allowed policy
output "writable_domains" {
  value = [
    for p in data.desec_token_policies.all.policies : p.domain
    if p.perm_write && p.domain != null
  ]
}
