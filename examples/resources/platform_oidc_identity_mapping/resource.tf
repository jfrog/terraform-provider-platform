resource "platform_oidc_identity_mapping" "my-github-oidc-user-identity-mapping" {
  name          = "my-github-oidc-user-identity-mapping"
  description   = "My GitHub OIDC user identity mapping"
  provider_name = "my-github-oidc-configuration"
  priority      = 1

  claims_json = jsonencode({
    "sub" = "repo:humpty/access-oidc-poc:ref:refs/heads/main",
    "workflow_ref" = "humpty/access-oidc-poc/.github/workflows/job.yaml@refs/heads/main"
  })

  token_spec = {
    username   = "my-user"
    scope      = "applied-permissions/user"
    audience   = "*@*"
    expires_in = 7200
  }
}

resource "platform_oidc_identity_mapping" "my-github-oidc-group-identity-mapping" {
  name          = "my-github-oidc-group-identity-mapping"
  description   = "My GitHub OIDC group identity mapping"
  provider_name = "my-github-oidc-configuration"
  priority      = 1

  claims_json = jsonencode({
    "sub" = "repo:humpty/access-oidc-poc:ref:refs/heads/main",
    "workflow_ref" = "humpty/access-oidc-poc/.github/workflows/job.yaml@refs/heads/main"
  })

  token_spec = {
    scope      = "applied-permissions/groups:\"readers\",\"my-group\""
    audience   = "jfrt@* jfac@* jfmc@* jfmd@* jfevt@* jfxfer@* jflnk@* jfint@* jfwks@*"
    expires_in = 7200
  }
}

resource "platform_oidc_identity_mapping" "my-github-oidc-project-roles-identity-mapping" {
  name          = "my-github-oidc-project-role-identity-mapping"
  description   = "My GitHub OIDC Project role identity mapping"
  provider_name = "my-github-oidc-configuration"
  priority      = 1

  claims_json = jsonencode({
    "sub" = "repo:humpty/access-oidc-poc:ref:refs/heads/main",
    "workflow_ref" = "humpty/access-oidc-poc/.github/workflows/job.yaml@refs/heads/main"
  })

  token_spec = {
    scope      = "applied-permissions/roles:my-project:\"Project Admin\",\"Developer\""
    audience   = "jfrt@* jfac@* jfmc@* jfmd@* jfevt@* jfxfer@* jflnk@* jfint@* jfwks@*"
    expires_in = 7200
  }

  project_key = "my-project"
}

resource "platform_oidc_identity_mapping" "my-github-oidc-username-pattern-identity-mapping" {
  name          = "my-github-oidc-username-pattern-identity-mapping"
  description   = "My GitHub OIDC username pattern identity mapping"
  provider_name = "my-github-oidc-configuration"
  priority      = 1

  claims_json = jsonencode({
    "sub" = "repo:humpty/access-oidc-poc:ref:refs/heads/main",
    "workflow_ref" = "humpty/access-oidc-poc/.github/workflows/job.yaml@refs/heads/main"
  })

  token_spec = {
    username_pattern = "{{user}}"
    audience         = "*@*"
    expires_in       = 7200
  }
}

resource "platform_oidc_identity_mapping" "my-github-oidc-groups-pattern-identity-mapping" {
  name          = "my-github-oidc-groups-pattern-identity-mapping"
  description   = "My GitHub OIDC groups pattern identity mapping"
  provider_name = "my-github-oidc-configuration"
  priority      = 1

  claims_json = jsonencode({
    "sub" = "repo:humpty/access-oidc-poc:ref:refs/heads/main",
    "workflow_ref" = "humpty/access-oidc-poc/.github/workflows/job.yaml@refs/heads/main"
  })

  token_spec = {
    groups_pattern = "{{group}}"
    audience       = "*@*"
    expires_in     = 7200
  }
}
