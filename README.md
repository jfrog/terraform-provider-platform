[![Terraform & OpenTofu Acceptance Tests](https://github.com/jfrog/terraform-provider-platform/actions/workflows/acceptance-tests.yml/badge.svg)](https://github.com/jfrog/terraform-provider-platform/actions/workflows/acceptance-tests.yml)

# Terraform Provider for JFrog Platform

## Quick Start

Create a new Terraform file with `platform` resource. Also see [sample.tf](./sample.tf):

### HCL Example

```terraform
# Required for Terraform 1.0 and later
terraform {
  required_providers {
    artifactory = {
      source  = "registry.terraform.io/jfrog/artifactory"
      version = "12.7.1"
    }
    platform = {
      source  = "registry.terraform.io/jfrog/platform"
      version = "2.2.0"
    }
  }
}

provider "artifactory" {
  // supply JFROG_URL and JFROG_ACCESS_TOKEN as env vars
}

provider "platform" {
  // supply JFROG_URL and JFROG_ACCESS_TOKEN as env vars
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
```

Initialize Terrform:
```sh
$ terraform init
```

Plan (or Apply):
```sh
$ terraform plan
```

Detailed documentation of the resource and attributes are on [Terraform Registry](https://registry.terraform.io/providers/jfrog/platform/latest/docs).

## License requirements:

This provider requires access to the APIs, which are only available in the _licensed_ pro and enterprise editions.
You can determine which license you have by accessing the following URL
`${host}/artifactory/api/system/licenses/`

You can either access it via api, or web browser - it does require admin level credentials, but it's one of the few APIs that will work without a license (side node: you can also install your license here with a `POST`)

```bash
curl -sL ${host}/artifactory/api/system/licenses/ | jq .
{
  "type" : "Enterprise Plus Trial",
  "validThrough" : "Jan 29, 2022",
  "licensedTo" : "JFrog Ltd"
}
```

## Versioning

In general, this project follows [semver](https://semver.org/) as closely as we can for tagging releases of the package. We've adopted the following versioning policy:

* We increment the **major version** with any incompatible change to functionality, including changes to the exported Go API surface or behavior of the API.
* We increment the **minor version** with any backwards-compatible changes to functionality.
* We increment the **patch version** with any backwards-compatible bug fixes.

## Contributors

See the [contribution guide](CONTRIBUTIONS.md).

## License

Copyright (c) 2025 JFrog.

Apache 2.0 licensed, see [LICENSE][LICENSE] file.

[LICENSE]: ./LICENSE
