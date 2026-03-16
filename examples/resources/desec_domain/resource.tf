# Create a DNS zone for example.com
resource "desec_domain" "example" {
  name = "example.com"
}

# Output the DNSSEC DS records for delegation at your registrar.
# These are flattened from all managed keys into a single list of DS record strings.
output "ds_records" {
  description = "DS records for DNSSEC delegation — enter these at your domain registrar."
  value       = flatten([for key in desec_domain.example.keys : key.ds if key.managed])
}

# Output the DNSKEY public key records for DNSSEC verification.
output "dnskeys" {
  description = "DNSKEY public key records for the domain."
  value       = [for key in desec_domain.example.keys : key.dnskey if key.managed]
}
