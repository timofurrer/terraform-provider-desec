# List all RRsets in a domain
data "desec_records" "all" {
  domain = "example.com"
}

output "all_record_types" {
  value = [for r in data.desec_records.all.records : "${r.subname}/${r.type}"]
}

# Filter by subname — all records for the www subdomain
data "desec_records" "www" {
  domain  = "example.com"
  subname = "www"
}

# Filter by record type — all A records regardless of subname
data "desec_records" "all_a" {
  domain = "example.com"
  type   = "A"
}

# Filter by both subname and type
data "desec_records" "www_a" {
  domain  = "example.com"
  subname = "www"
  type    = "A"
}
