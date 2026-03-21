# Look up a specific RRset
data "desec_rrset" "www_a" {
  domain  = "example.com"
  subname = "www"
  type    = "A"
}

output "www_ips" {
  value = data.desec_rrset.www_a.rdata
}

output "www_ttl" {
  value = data.desec_rrset.www_a.ttl
}

# Look up the MX RRset at the zone apex
data "desec_rrset" "mx" {
  domain  = "example.com"
  subname = "@"
  type    = "MX"
}
