# Convert a unicode domain name to Punycode before registering it with deSEC.
# result: "xn--mnchen-3ya.de"
resource "desec_domain" "idn" {
  name = provider::desec::to_punycode("münchen.de")
}

# Standalone conversion — useful for outputs or local values.
output "punycoded" {
  value = provider::desec::to_punycode("münchen.de")
}
