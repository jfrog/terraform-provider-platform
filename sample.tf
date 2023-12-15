terraform {
  required_providers {
    platform = {
      source  = "registry.terraform.io/jfrog/platform"
      version = "1.0.3"
    }

    artifactory = {
      source  = "registry.terraform.io/jfrog/artifactory"
      version = "10.0.0"
    }
  }
}

variable "jfrog_url" {
  type = string
  default = "https://myinstance.jfrog.io"
}

provider "platform" {
  url = "${var.jfrog_url}"
  // supply JFROG_ACCESS_TOKEN as env var
}

resource "artifactory_local_generic_repository" "my-generic-local" {
  key = "my-generic-local"
}

resource "platform_workers_service" "my-workers-service" {
  key         = "my-workers-service"
  enabled     = true
  description = "My workers service"
  source_code = <<EOT
export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => {
  console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping'));
  console.log(await axios.get('https://my.external.resource'));
  return {
    status: 'DOWNLOAD_PROCEED',
    message: 'proceed',
  }
}
EOT
  action      = "BEFORE_DOWNLOAD"

  filter_criteria = {
    artifact_filter_criteria = {
      repo_keys = [artifactory_local_generic_repository.my-generic-local.key]
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
