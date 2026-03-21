# List all DNS RRsets for a domain
list "desec_rrset" "all" {
  provider = desec
  config {
    domain = "example.com"
  }
}

# List only A RRsets
list "desec_rrset" "a_records" {
  provider = desec
  config {
    domain = "example.com"
    type   = "A"
  }
}

# List RRsets for a specific subdomain
list "desec_rrset" "www" {
  provider = desec
  config {
    domain  = "example.com"
    subname = "www"
  }
}
