# Parse a DNSKEY record from a domain's DNSSEC key material.
data "desec_domain" "example" {
  name = "example.com"
}

locals {
  dnskey = provider::desec::parse_dnskey(data.desec_domain.example.keys[0].dnskey)
}

output "flags" {
  value = local.dnskey.flags
}

output "algorithm" {
  value = local.dnskey.algorithm
}

output "public_key" {
  value = local.dnskey.public_key
}
