# List all domains for the account
list "desec_domain" "all" {
  provider = desec
}

# List domains that own a specific fully-qualified domain name
list "desec_domain" "owns" {
  provider = desec
  config {
    owns_qname = "sub.example.com."
  }
}
