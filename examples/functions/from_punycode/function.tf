# Decode a Punycode domain name back to its unicode form for display.
# result: "münchen.de"
output "unicode_domain" {
  value = provider::desec::from_punycode("xn--mnchen-3ya.de")
}

# Round-trip: encode then decode returns the original unicode name.
output "roundtrip" {
  value = provider::desec::from_punycode(provider::desec::to_punycode("münchen.de"))
}
