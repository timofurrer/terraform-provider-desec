# Discover all domains in the account
list "desec_domain" "all" {
  provider = desec
}

# Discover all DNS records for a domain
list "desec_record" "all" {
  provider = desec
  config {
    domain = "example.com"
  }
}
