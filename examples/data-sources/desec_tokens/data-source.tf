# Retrieve all tokens in the account
data "desec_tokens" "all" {}

# Output all token names
output "token_names" {
  value = [for t in data.desec_tokens.all.tokens : t.name]
}

# Output names of all currently valid tokens
output "valid_token_names" {
  value = [for t in data.desec_tokens.all.tokens : t.name if t.is_valid]
}
