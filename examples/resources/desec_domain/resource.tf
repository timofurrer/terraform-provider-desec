# Create a DNS zone for example.com
resource "desec_domain" "example" {
  name = "example.com"
}

# Output the DNSSEC DS records for delegation at your registrar
output "ds_records" {
  value = [for key in desec_domain.example.keys : key.ds if key.managed]
}
