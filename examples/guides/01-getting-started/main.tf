provider "desec" {
  api_token = var.desec_api_token
}

variable "desec_api_token" {
  type      = string
  sensitive = true
}

# Step 1: Register a domain with deSEC
resource "desec_domain" "example" {
  name = "example.com"
}

# Step 2: Retrieve the nameservers assigned by deSEC
data "desec_rrset" "nameservers" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "NS"
}

output "nameservers" {
  description = "Enter these nameservers at your domain registrar."
  value       = data.desec_rrset.nameservers.rdata
}

# Step 2b: Retrieve DNSSEC DS records for your registrar
output "dnssec_ds_records" {
  description = "DS records for DNSSEC delegation — enter these at your domain registrar."
  value       = flatten([for key in desec_domain.example.keys : key.ds if key.managed])
}

output "dnssec_dnskeys" {
  description = "DNSKEY public key records for DNSSEC verification."
  value       = [for key in desec_domain.example.keys : key.dnskey if key.managed]
}

# Step 3: Create individual DNS RRsets
resource "desec_rrset" "www_a" {
  domain  = desec_domain.example.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  rdata   = ["203.0.113.10"]
}

resource "desec_rrset" "mx" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "MX"
  ttl     = 3600
  rdata   = ["10 mail.example.com."]
}

resource "desec_rrset" "spf" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "TXT"
  ttl     = 3600
  rdata   = ["\"v=spf1 mx ~all\""]
}

# Step 4: Bulk RRset management with desec_records
resource "desec_domain" "bulk" {
  name = "bulk.example.com"
}

resource "desec_records" "bulk" {
  domain    = desec_domain.bulk.name
  exclusive = true

  zonefile = <<-ZONE
    bulk.example.com.      3600 IN A     203.0.113.10
    bulk.example.com.      3600 IN AAAA  2001:db8::1
    www.bulk.example.com.  3600 IN A     203.0.113.10
    bulk.example.com.      3600 IN MX    10 mail.example.com.
    bulk.example.com.      3600 IN TXT   "v=spf1 mx ~all"
  ZONE
}

# Internationalized domain names (IDN): use to_punycode() for unicode domains
resource "desec_domain" "idn" {
  name = provider::desec::to_punycode("münchen.de")
}
