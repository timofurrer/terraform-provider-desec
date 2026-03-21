# Publish an OpenPGP public key as a DANE OPENPGPKEY DNS record (RFC 7929).
locals {
  dane = provider::desec::openpgpkey_dane("hugh@example.com", file("hugh.gpg.base64"))
}

resource "desec_rrset" "openpgpkey" {
  domain  = local.dane.domain
  subname = local.dane.subname
  type    = local.dane.type
  ttl     = 3600
  rdata   = [local.dane.rdata]
}
