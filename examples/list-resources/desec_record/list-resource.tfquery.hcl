# List all RRsets within a domain
list "desec_record" "all" {
  provider = desec
  config {
    domain = "example.com"
  }
}

# List only A records within a domain
list "desec_record" "a_records" {
  provider = desec
  config {
    domain = "example.com"
    type   = "A"
  }
}

# List RRsets for a specific subdomain
list "desec_record" "www" {
  provider = desec
  config {
    domain  = "example.com"
    subname = "www"
  }
}
