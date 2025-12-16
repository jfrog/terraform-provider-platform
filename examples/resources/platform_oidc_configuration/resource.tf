resource "platform_oidc_configuration" "my-github-oidc-configuration" {
  name          = "my-github-oidc-configuration"
  description   = "My GitHub OIDC configuration"
  issuer_url    = "https://token.actions.githubusercontent.com"
  provider_type = "GitHub"
  organization  = "jfrog"
  audience      = "jfrog-github"
}

resource "platform_oidc_configuration" "my-github-oidc-enterprise-configuration" {
  name          = "my-github-oidc-enterprise-configuration"
  description   = "My GitHub OIDC enterprise configuration"
  issuer_url    = "https://token.actions.githubusercontent.com/jfrog"
  provider_type = "GitHubEnterprise"
  organization  = "jfrog"
  audience      = "jfrog-github"
}

resource "platform_oidc_configuration" "my-generic-oidc-configuration" {
  name          = "my-generic-oidc-configuration"
  description   = "My generic OIDC configuration"
  issuer_url    = "https://tempurl.org"
  provider_type = "generic"
  audience      = "jfrog-generic"
}


resource "platform_oidc_configuration" "my-azure-oidc-configuration" {
  name              = "{{ .name }}"
  description       = "My Azure OIDC configuration"
  issuer_url        = "https://sts.windows.net/your-tenant-id/"
  provider_type     = "Azure"
  audience          = "azure-audience"
  use_default_proxy = true
}