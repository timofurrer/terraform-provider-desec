# Look up an existing domain by name
data "desec_domain" "example" {
  name = "example.com"
}

# Use the minimum TTL in other resources
output "minimum_ttl" {
  value = data.desec_domain.example.minimum_ttl
}

# Access DNSSEC key information
output "ds_records" {
  value = [for key in data.desec_domain.example.keys : key.ds if key.managed]
}
