# Look up a specific token policy by token UUID and policy UUID
data "desec_token_policy" "existing" {
  token_id = "3a6b94b5-d20e-40bd-a7cc-521f5c79fab3"
  id       = "7aed3f71-bc81-4f7e-90ae-8f0df0d1c211"
}

output "policy_perm_write" {
  value = data.desec_token_policy.existing.perm_write
}
