# Look up an existing token by its UUID
data "desec_token" "existing" {
  id = "3a6b94b5-d20e-40bd-a7cc-521f5c79fab3"
}

output "token_name" {
  value = data.desec_token.existing.name
}

output "token_valid" {
  value = data.desec_token.existing.is_valid
}
