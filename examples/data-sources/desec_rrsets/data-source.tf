# List all RRsets in a domain
data "desec_rrsets" "all" {
  domain = "example.com"
}

output "all_record_types" {
  value = [for r in data.desec_rrsets.all.rrsets : "${r.subname}/${r.type}"]
}

# Filter by subname — all RRsets for the www subdomain
data "desec_rrsets" "www" {
  domain  = "example.com"
  subname = "www"
}

# Filter by record type — all A RRsets regardless of subname
data "desec_rrsets" "all_a" {
  domain = "example.com"
  type   = "A"
}

# Filter by both subname and type
data "desec_rrsets" "www_a" {
  domain  = "example.com"
  subname = "www"
  type    = "A"
}
