# Discover all domains in the account
list "desec_domain" "all" {
  provider = desec
}

# Discover all DNS RRsets for a domain
list "desec_rrset" "all" {
  provider = desec
  config {
    domain = "example.com"
  }
}
