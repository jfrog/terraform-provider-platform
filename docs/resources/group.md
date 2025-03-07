---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "platform_group Resource - terraform-provider-platform"
subcategory: ""
description: |-
  Provides a group resource to create and manage groups, and manages membership. A group represents a role and is used with RBAC (Role-Based Access Control) rules. See JFrog documentation https://jfrog.com/help/r/jfrog-platform-administration-documentation/create-and-edit-groups for more details.
---

# platform_group (Resource)

Provides a group resource to create and manage groups, and manages membership. A group represents a role and is used with RBAC (Role-Based Access Control) rules. See [JFrog documentation](https://jfrog.com/help/r/jfrog-platform-administration-documentation/create-and-edit-groups) for more details.

## Example Usage

```terraform
resource "platform_group" "my-group" {
  name = "my-group"
  description = "My group"
  external_id = "My Azure ID"
  auto_join = true
  admin_privileges = false
  members = [
    "admin"
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the group.

### Optional

- `admin_privileges` (Boolean) Any users added to this group will automatically be assigned with admin privileges in the system.
- `auto_join` (Boolean) When this parameter is set, any new users defined in the system are automatically assigned to this group.
- `description` (String) A description for the group.
- `external_id` (String) New external group ID used to configure the corresponding group in Azure AD.
- `members` (Set of String, Deprecated) List of users assigned to the group.
- `use_group_members_resource` (Boolean) When set to `true`, this resource will ignore the `members` attributes and allow memberships to be managed by `platform_group_members` resource instead. Default value is `true`.

### Read-Only

- `realm` (String) The realm for the group.
- `realm_attributes` (String) The realm for the group.

## Import

Import is supported using the following syntax:

```shell
terraform import platform_group.my-group my-group
```
