terraform {
  required_providers {
    desec = {
      source = "timofurrer/desec"
    }
  }
}

# NOTE: set TF_CLI_CONFIG_FILE=$(pwd)/.terraformrc

provider "desec" {
}

resource "desec_domain" "example" {
  name = "example-desec-provider.com"
}

# Create an A record for www.example.com
resource "desec_record" "www_a" {
  domain  = desec_domain.example.name
  subname = "www"
  type    = "A"
  ttl     = 3600
  records = ["1.2.3.4", "5.6.7.8"]
}

# Create an AAAA record for www.example.com
resource "desec_record" "www_aaaa" {
  domain  = desec_domain.example.name
  subname = "www"
  type    = "AAAA"
  ttl     = 3600
  records = ["2001:db8::1"]
}

# Create an MX record at the zone apex
resource "desec_record" "mx" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "MX"
  ttl     = 3600
  records = ["10 mail.example.com.", "20 mail2.example.com."]
}

# Create a TXT record for SPF
resource "desec_record" "spf" {
  domain  = desec_domain.example.name
  subname = "@"
  type    = "TXT"
  ttl     = 3600
  records = ["\"v=spf1 include:_spf.example.com ~all\""]
}

# Create a CNAME record
resource "desec_record" "blog_cname" {
  domain  = desec_domain.example.name
  subname = "blog"
  type    = "CNAME"
  ttl     = 3600
  records = ["www.example.com."]
}
