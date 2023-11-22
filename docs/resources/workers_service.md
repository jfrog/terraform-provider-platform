---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "platform_workers_service Resource - terraform-provider-platform"
subcategory: ""
description: |-
  Provides a JFrog Workers Service https://jfrog.com/help/r/jfrog-platform-administration-documentation/workers-service resource. This can be used to create and manage Workers Service.
  !>JFrog Workers Service is only available for JFrog Cloud customers to use free of charge during the beta period. The API may not be backward compatible after the beta period is over. Be aware of this caveat when you create workers during this period.
---

# platform_workers_service (Resource)

Provides a JFrog [Workers Service](https://jfrog.com/help/r/jfrog-platform-administration-documentation/workers-service) resource. This can be used to create and manage Workers Service.

!>JFrog Workers Service is only available for JFrog Cloud customers to use free of charge during the beta period. The API may not be backward compatible after the beta period is over. Be aware of this caveat when you create workers during this period.

## Example Usage

```terraform
resource "platform_workers_service" "my-workers-service" {
  key         = "my-workers-service"
  enabled     = true
  description = "My workers service"
  source_code = "export default async (context: PlatformContext, data: BeforeDownloadRequest): Promise<BeforeDownloadResponse> => { console.log(await context.clients.platformHttp.get('/artifactory/api/system/ping')); console.log(await axios.get('https://my.external.resource')); return { status: 'DOWNLOAD_PROCEED', message: 'proceed', } }"
  action      = "BEFORE_DOWNLOAD"

  filter_criteria = {
    artifact_filter_criteria = {
      repo_keys = ["my-repo-key"]
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

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `action` (String) The worker action with which the worker is associated. Valid values: BEFORE_DOWNLOAD, AFTER_DOWNLOAD, BEFORE_UPLOAD, AFTER_CREATE, AFTER_BUILD_INFO_SAVE, AFTER_MOVE
- `enabled` (Boolean) Whether to enable the worker immediately after creation.
- `filter_criteria` (Attributes) Defines the repositories to be used or excluded. (see [below for nested schema](#nestedatt--filter_criteria))
- `key` (String) The unique ID of the worker.
- `source_code` (String) The worker script in TypeScript or JavaScript.

### Optional

- `description` (String) Description of the worker.
- `secrets` (Attributes Set) The secrets to be added to the worker. (see [below for nested schema](#nestedatt--secrets))

<a id="nestedatt--filter_criteria"></a>
### Nested Schema for `filter_criteria`

Required:

- `artifact_filter_criteria` (Attributes) (see [below for nested schema](#nestedatt--filter_criteria--artifact_filter_criteria))

<a id="nestedatt--filter_criteria--artifact_filter_criteria"></a>
### Nested Schema for `filter_criteria.artifact_filter_criteria`

Required:

- `repo_keys` (Set of String) Defines which repositories are used when an action event occurs to trigger the worker.

Optional:

- `exclude_patterns` (Set of String) Define patterns to for all repository paths for repositories to be excluded in the repoKeys. Defines those repositories that do not trigger the worker.
- `include_patterns` (Set of String) Define patterns to match all repository paths for repositories identified in the repoKeys. Defines those repositories that trigger the worker.



<a id="nestedatt--secrets"></a>
### Nested Schema for `secrets`

Required:

- `key` (String) The name of the secret.
- `value` (String) The name of the secret.