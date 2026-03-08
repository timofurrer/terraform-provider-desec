# List all domains in the account
data "desec_domains" "all" {}

output "domain_names" {
  value = [for d in data.desec_domains.all.domains : d.name]
}

# Find the authoritative domain for a specific DNS name
data "desec_domains" "responsible" {
  owns_qname = "www.example.com"
}

output "responsible_domain" {
  value = length(data.desec_domains.responsible.domains) > 0 ? data.desec_domains.responsible.domains[0].name : null
}
