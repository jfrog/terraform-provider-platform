terraform {
  required_providers {
    artifactory = {
      source  = "jfrog/artifactory"
      version = "10.5.1"
    }
    platform = {
      source  = "jfrog/platform"
      version = "1.6.0"
    }
  }
}

variable "jfrog_url" {
  type = string
  default = "http://localhost:8081"
}

provider "artifactory" {
  url = "${var.jfrog_url}"
  // supply JFROG_ACCESS_TOKEN as env var
}

provider "platform" {
  url = "${var.jfrog_url}"
  // supply JFROG_ACCESS_TOKEN as env var
}

resource "platform_global_role" "my-global-role" {
  name         = "my-global-role"
  description  = "Test description"
  type         = "CUSTOM_GLOBAL"
  environments = ["DEV"]
  actions      = ["READ_REPOSITORY", "READ_BUILD"]
}

resource "artifactory_local_generic_repository" "my-generic-local" {
  key = "my-generic-local"
}

resource "platform_workers_service" "my-workers-service" {
  key         = "my-workers-service"
  enabled     = true
  description = "My workers service"
  source_code = "export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { status: 'DOWNLOAD_PROCEED', message: 'proceed', } }"
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