# Look up a specific RRset
data "desec_record" "www_a" {
  domain  = "example.com"
  subname = "www"
  type    = "A"
}

output "www_ips" {
  value = data.desec_record.www_a.records
}

output "www_ttl" {
  value = data.desec_record.www_a.ttl
}

# Look up the MX record at the zone apex
data "desec_record" "mx" {
  domain  = "example.com"
  subname = "@"
  type    = "MX"
}
