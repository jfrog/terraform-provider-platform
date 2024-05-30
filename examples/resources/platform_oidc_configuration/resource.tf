resource "platform_oidc_configuration" "my-github-oidc-configuration" {
  name          = "my-github-oidc-configuration"
  description   = "My GitHub OIDC configuration"
  issuer_url    = "https://token.actions.githubusercontent.com"
  provider_type = "GitHub"
  audience      = "jfrog-github"
}

resource "platform_oidc_configuration" "my-generic-oidc-configuration" {
  name          = "my-generic-oidc-configuration"
  description   = "My generic OIDC configuration"
  issuer_url    = "https://tempurl.org"
  provider_type = "generic"
  audience      = "jfrog-generic"
}