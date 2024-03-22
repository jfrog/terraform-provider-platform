resource "platform_oidc_identity_mapping" "my-github-oidc-identity-mapping" {
  name          = "my-github-oidc-identity-mapping"
  description   = "My GitHub OIDC identity mapping"
  provider_name = "my-github-oidc-configuration"
  priority      = 1

  claims = {
    sub          = "repo:humpty/access-oidc-poc:ref:refs/heads/main"
    workflow_ref = "humpty/access-oidc-poc/.github/workflows/job.yaml@refs/heads/main"
  }

  token_spec = {
    username   = "my-user"
    scope      = "applied-permissions/user"
    audience   = "*@*"
    expires_in = 7200
  }
}