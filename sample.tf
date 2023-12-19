terraform {
  required_providers {
    platform = {
      source  = "registry.terraform.io/jfrog/platform"
      version = "1.0.2"
    }
  }
}

variable "jfrog_url" {
  type = string
  default = "https://partnership.jfrog.io"
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
