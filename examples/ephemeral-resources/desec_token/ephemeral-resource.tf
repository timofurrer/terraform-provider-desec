ephemeral "desec_token" "example" {
  name               = "ci-deploy-token"
  perm_create_domain = false
  keep_on_close      = false
}

output "deploy_token" {
  value     = ephemeral.desec_token.example.token
  sensitive = true
}
