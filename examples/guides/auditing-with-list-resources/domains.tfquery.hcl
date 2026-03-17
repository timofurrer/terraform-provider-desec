# List all domains in the account
list "desec_domain" "all" {
  provider = desec
}

# Find which domain owns a specific fully-qualified domain name
list "desec_domain" "owns" {
  provider = desec
  config {
    owns_qname = "sub.example.com."
  }
}
