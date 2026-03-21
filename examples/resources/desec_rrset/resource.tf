# Create a DNS zone first
resource "desec_domain" "example" {
  name = "example.com"
}

# Create an A RRset for www.example.com
resource "desec_rrset" "www_a" {
  domain  = desec_domain.example.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  rdata   = ["1.2.3.4", "5.6.7.8"]
}

# Create an AAAA RRset for www.example.com
resource "desec_rrset" "www_aaaa" {
  domain  = desec_domain.example.name
  subname = "www"
  type    = "AAAA"
  ttl     = 3600
  rdata   = ["2001:db8::1"]
}

# Create an MX RRset at the zone apex
resource "desec_rrset" "mx" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "MX"
  ttl     = 3600
  rdata   = ["10 mail.example.com.", "20 mail2.example.com."]
}

# Create a TXT RRset for SPF
resource "desec_rrset" "spf" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "TXT"
  ttl     = 3600
  rdata   = ["\"v=spf1 include:_spf.example.com ~all\""]
}

# Create a CNAME RRset
resource "desec_rrset" "blog_cname" {
  domain  = desec_domain.example.name
  subname = "blog"
  type    = "CNAME"
  ttl     = 3600
  rdata   = ["www.example.com."]
}

# Publish an OpenPGP public key as a DANE OPENPGPKEY record (RFC 7929)
locals {
  dane = provider::desec::openpgpkey_dane("hugh@example.com", file("hugh.gpg.base64"))
}

resource "desec_record" "openpgpkey" {
  domain  = local.dane.domain
  subname = local.dane.subname
  type    = local.dane.type
  ttl     = 3600
  records = [local.dane.rdata]
}
