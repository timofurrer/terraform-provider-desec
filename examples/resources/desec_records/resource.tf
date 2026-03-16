# --- Mode A: Manage records via a BIND zone file ---

resource "desec_domain" "example" {
  name = "example.com"
}

resource "desec_records" "from_zonefile" {
  domain = desec_domain.example.name

  # Inline zone file. Use file() or templatefile() to load from disk:
  #   zonefile = file("${path.module}/example.com.zone")
  zonefile = <<-ZONE
    example.com.      3600 IN A    203.0.113.10
    example.com.      3600 IN AAAA 2001:db8::1
    www.example.com.  3600 IN A    203.0.113.10
    example.com.      3600 IN MX   10 mail.example.com.
    mail.example.com. 3600 IN A    203.0.113.20
    example.com.      3600 IN TXT  "v=spf1 mx ~all"
  ZONE

  depends_on = [desec_domain.example]
}

# --- Mode B: Manage records via structured RRsets ---

resource "desec_records" "from_records" {
  domain = desec_domain.example.name

  records = [
    {
      subname = ""
      type    = "A"
      ttl     = 3600
      records = ["203.0.113.10"]
    },
    {
      subname = ""
      type    = "AAAA"
      ttl     = 3600
      records = ["2001:db8::1"]
    },
    {
      subname = "www"
      type    = "A"
      ttl     = 3600
      records = ["203.0.113.10"]
    },
    {
      subname = ""
      type    = "MX"
      ttl     = 3600
      records = ["10 mail.example.com."]
    },
    {
      subname = "mail"
      type    = "A"
      ttl     = 3600
      records = ["203.0.113.20"]
    },
    {
      subname = ""
      type    = "TXT"
      ttl     = 3600
      records = ["\"v=spf1 mx ~all\""]
    },
  ]

  depends_on = [desec_domain.example]
}

# Reference the computed attributes:
#   desec_records.from_zonefile.records  — structured RRsets (when using Mode A)
#   desec_records.from_records.zonefile  — canonical zone file (when using Mode B)

# Import example:
#   terraform import desec_records.from_zonefile example.com
