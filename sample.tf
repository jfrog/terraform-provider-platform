terraform {
  required_providers {
    platform = {
      source  = "jfrog/platform"
      version = "2.2.4"
    }
  }
}

variable "jfrog_url" {
  type = string
  default = "https://my.jfrog.io"
}

provider "platform" {
  url = "${var.jfrog_url}"
  // supply JFROG_ACCESS_TOKEN as env var
}

resource "platform_workers_service" "my-workers-service" {
  key         = "my-workers-service"
  enabled     = true
  description = "My workers service"
  source_code = <<EOT
export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => {
  console.log(await context.clients.platform Http.get('/artifactory/api/system/ping'));
  console.log(await axios.get('https://my.external.resource'));
  return {
    status: 'DOWNLOAD_PROCEED',
    message: 'proceed',
  }
}
EOT
  action = "BEFORE_DOWNLOAD"

  filter_criteria = {
    artifact_filter_criteria = {
      repo_keys = ["my-generic-local"]
    }
  }

  secrets = [
    {
      key   = "my-secret-key-1"
      value = "my-secret-value-1"
    },
    {
      key   = "my-secret-key-2"
      value = "my-secret-value-2"
    }
  ]
}

resource "platform_permission" "my-permission" {
  name = "my-permission-name"

  artifact = {
    actions = {
      users = [
        {
          name = "my-user"
          permissions = ["READ", "WRITE"]
        }
      ]
    }

    targets = [
      {
        name = "my-docker-local"
        include_patterns = ["**"]
      },
      {
        name = "ALL-LOCAL"
        include_patterns = ["**", "*.js"]
      },
      {
        name = "ALL-REMOTE"
        include_patterns = ["**", "*.js"]
      },
      {
        name = "ALL-DISTRIBUTION"
        include_patterns = ["**", "*.js"]
      }
    ]
  }

  build = {
    targets = [
      {
        name = "artifactory-build-info"
        include_patterns = ["**"]
        exclude_patterns = ["*.js"]
      }
    ] 
  }

  release_bundle = {
    actions = {
      users = [
        {
          name = "my-user"
          permissions = ["READ", "WRITE"]
        }
      ]

      groups = [
        {
          name = "my-group"
          permissions = ["READ", "ANNOTATE"]
        }
      ]
    }

    targets = [
      {
        name = "release-bundle"
        include_patterns = ["**"]
      }
    ]
  }

  destination = {
    actions = {
      groups = [
        {
          name = "my-group"
          permissions = ["READ", "ANNOTATE"]
        }
      ]
    }

    targets = [
      {
        name = "*"
        include_patterns = ["**"]
      }
    ]
  }

  pipeline_source = {
    actions = {
      groups = [
        {
          name = "my-group"
          permissions = ["READ", "ANNOTATE"]
        }
      ]
    }

    targets = [
      {
        name = "*"
        include_patterns = ["**"]
      }
    ]
  }
}


# GitHub Actions OIDC provider
resource "platform_oidc_configuration" "github" {
  name          = "tf-github"
  issuer_url    = "https://token.actions.githubusercontent.com"
  provider_type = "GitHub"
  organization  = "my-github-org"
  audience      = "jfrog-platform"
}

# Azure OIDC provider with azure_app_id (new in 2.2.11)
resource "platform_oidc_configuration" "azure" {
  name          = "tf-azure"
  issuer_url    = "https://login.microsoftonline.com/my-tenant-id/v2.0"
  provider_type = "Azure"
  azure_app_id  = "00000000-0000-0000-0000-000000000000"
}

# username_pattern + groups_pattern together (both now allowed — bug fix in 2.2.11)
resource "platform_oidc_identity_mapping" "username_and_groups_pattern" {
  name          = "tf-username-and-groups-pattern"
  provider_name = platform_oidc_configuration.generic_global.name
  priority      = 40
  claims_json   = jsonencode({ sub = "ci-runner" })

  token_spec = {
    username_pattern = "{sub}"
    groups_pattern   = "{groups}"
    audience         = "*@*"
    expires_in       = 60
  }
}

# self_revocable token (new in 2.2.11)
resource "platform_oidc_identity_mapping" "self_revocable" {
  name          = "tf-self-revocable"
  provider_name = platform_oidc_configuration.generic_global.name
  priority      = 50
  claims_json   = jsonencode({ sub = "ci-runner" })

  token_spec = {
    username       = "admin"
    scope          = "applied-permissions/user"
    audience       = "*@*"
    expires_in     = 3600
    self_revocable = true
  }
}

# Project-scoped identity mapping — roles scope
resource "platform_oidc_identity_mapping" "project_roles" {
  name          = "tf-project-roles"
  provider_name = platform_oidc_configuration.generic_global.name
  priority      = 10
  project_key   = "myproject"
  claims_json   = jsonencode({ sub = "project-developer" })

  token_spec = {
    username   = "admin"
    scope      = "applied-permissions/roles:myproject:\"Developer\""
    audience   = "*@*"
    expires_in = 120
  }
}
