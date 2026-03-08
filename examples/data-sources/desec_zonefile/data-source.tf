# Export a domain as a zonefile
data "desec_zonefile" "example" {
  name = "example.com"
}
