# Parse a DS record from a domain's DNSSEC key material.
data "desec_domain" "example" {
  name = "example.com"
}

locals {
  ds = provider::desec::parse_ds(data.desec_domain.example.keys[0].ds[0])
}

output "key_tag" {
  value = local.ds.key_tag
}

output "digest_type" {
  value = local.ds.digest_type
}

output "digest" {
  value = local.ds.digest
}
