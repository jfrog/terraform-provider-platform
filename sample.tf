terraform {
  required_providers {
    platform = {
      source  = "jfrog/platform"
      version = "1.19.2"
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