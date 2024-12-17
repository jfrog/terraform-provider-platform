---
layout: ""
page_title: "JFrog Platform Provider"
description: |-
  The JFrog Platform provider provides resources to interact with features from JFrog platform.
---

# JFrog Platform Provider

The [JFrog](https://jfrog.com/) Platform provider is used to interact with the features from [JFrog Platform REST API](https://jfrog.com/help/r/jfrog-rest-apis/jfrog-platform-rest-apis). The provider needs to be configured with the proper credentials before it can be used.

Links to documentation for specific resources can be found in the table of contents to the left.

This provider requires access to JFrog Platform APIs, which are only available in the _licensed_ pro and enterprise editions. You can determine which license you have by accessing the following URL `${host}/artifactory/api/system/licenses/`

You can either access it via API, or web browser - it requires admin level credentials.

```bash
curl -sL ${host}/artifactory/api/system/licenses/ | jq .
{
  "type" : "Enterprise Plus Trial",
  "validThrough" : "Jan 29, 2022",
  "licensedTo" : "JFrog Ltd"
}
```

## Example Usage

```terraform
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
```

## Authentication

The JFrog Platform provider supports for the following types of authentication:
* Scoped token
* Terraform Cloud OIDC provider

### Scoped Token

JFrog scoped tokens may be used via the HTTP Authorization header by providing the `access_token` field to the provider block. Getting this value from the environment is supported with the `JFROG_ACCESS_TOKEN` environment variable.

Usage:
```terraform
provider "platform" {
  url = "my.jfrog.io"
  access_token = "abc...xy"
}
```

### Terraform Cloud OIDC Provider

If you are using this provider on Terraform Cloud and wish to use dynamic credentials instead of static access token for authentication with JFrog platform, you can leverage Terraform as the OIDC provider.

To setup dynamic credentials, follow these steps:
1. Configure Terraform Cloud as a generic OIDC provider
2. Set environment variable in your Terraform Workspace
3. Setup Terraform Cloud in your configuration

During the provider start up, if it finds env var `TFC_WORKLOAD_IDENTITY_TOKEN` it will use this token with your JFrog instance to exchange for a short-live access token. If that is successful, the provider will the access token for all subsequent API requests with the JFrog instance.

#### Configure Terraform Cloud as generic OIDC provider

Follow [confgure an OIDC integration](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-an-oidc-integration). Enter a name for the provider, e.g. `terraform-cloud`. Use `https://app.terraform.io` for "Provider URL". Choose your own value for "Audience", e.g. `jfrog-terraform-cloud`.

Then [configure an identity mapping](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-identity-mappings) with appropriate "Claims JSON" (e.g. `aud`, `sub` at minimum. See [Terraform Workload Identity - Configuring Trust with your Cloud Platform](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/workload-identity-tokens#configuring-trust-with-your-cloud-platform)), and select the "Token scope", "User", and "Service" as desired.

#### Set environment variable in your Terraform Workspace

In your workspace, add an environment variable `TFC_WORKLOAD_IDENTITY_AUDIENCE` with audience value (e.g. `jfrog-terraform-cloud`) from JFrog OIDC integration above. See [Manually Generating Workload Identity Tokens](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/manual-generation) for more details.

When a run starts on Terraform Cloud, it will create a workload identity token with the specified audience and assigns it to the environment variable `TFC_WORKLOAD_IDENTITY_TOKEN` for the provider to consume.

See [Generating Multiple Tokens](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/manual-generation#generating-multiple-tokens) on HCP Terraform for more details on using different tokens.

#### Setup Terraform Cloud in your configuration

Add `cloud` block to `terraform` block, and add `oidc_provider_name` attribute (from JFrog OIDC integration) to provider block:

```terraform
terraform {
  cloud {
    organization = "my-org"
    workspaces {
      name = "my-workspace"
    }
  }

  required_providers {
    platform = {
      source  = "jfrog/platform"
      version = "1.6.1"
    }
  }
}

provider "platform" {
  url = "https://myinstance.jfrog.io"
  oidc_provider_name = "terraform-cloud"
  tfc_credential_tag_name = "JFROG"
}
```

**Note:** Ensure `access_token` attribute is not set

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `access_token` (String, Sensitive) This is a access token that can be given to you by your admin under `Platform Configuration -> User Management -> Access Tokens`. This can also be sourced from the `JFROG_ACCESS_TOKEN` environment variable.
- `oidc_provider_name` (String) OIDC provider name. See [Configure an OIDC Integration](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-an-oidc-integration) for more details.
- `tfc_credential_tag_name` (String) Terraform Cloud Workload Identity Token tag name. Use for generating multiple TFC workload identity tokens. When set, the provider will attempt to use env var with this tag name as suffix. **Note:** this is case sensitive, so if set to `JFROG`, then env var `TFC_WORKLOAD_IDENTITY_TOKEN_JFROG` is used instead of `TFC_WORKLOAD_IDENTITY_TOKEN`. See [Generating Multiple Tokens](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/manual-generation#generating-multiple-tokens) on HCP Terraform for more details.
- `url` (String) JFrog Platform URL. This can also be sourced from the `JFROG_URL` environment variable.
