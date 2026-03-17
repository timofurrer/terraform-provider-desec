# List all API tokens for the account
list "desec_token" "all" {
  provider = desec
}

# List all policies for a specific token
list "desec_token_policy" "all" {
  provider = desec
  config {
    token_id = "00000000-0000-0000-0000-000000000000"
  }
}
