---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "platform_http_sso_settings Resource - terraform-provider-platform"
subcategory: ""
description: |-
  Provides a JFrog HTTP SSO Settings https://jfrog.com/help/r/jfrog-platform-administration-documentation/http-sso resource. This allows you to reuse existing HTTP-based SSO infrastructures with the JFrog Platform Unit (JPD), such as the SSO modules offered by Apache HTTPd.
---

# platform_http_sso_settings (Resource)

Provides a JFrog [HTTP SSO Settings](https://jfrog.com/help/r/jfrog-platform-administration-documentation/http-sso) resource. This allows you to reuse existing HTTP-based SSO infrastructures with the JFrog Platform Unit (JPD), such as the SSO modules offered by Apache HTTPd.

## Example Usage

```terraform
resource "platform_http_sso_settings" "my-http-sso-settings" {
	proxied                      = true
	auto_create_user             = true
	allow_user_to_access_profile = true
	remote_user_request_variable = "MY_REMOTE_USER"
	sync_ldap_groups             = false
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `proxied` (Boolean) When set, Artifactory trusts incoming requests and reuses the remote user originally set on the request by the SSO of the HTTP server. This is useful if you want to use existing enterprise SSO integrations, such as the powerful authentication schemes provided by Apache (mod_auth_ldap, mod_auth_ntlm, mod_auth_kerb, etc.). When Artifactory is deployed as a webapp on Tomcat behind Apache: If using mod_jk, be sure to use the `JkEnvVar REMOTE_USER` directive in Apache's configuration.

### Optional

- `allow_user_to_access_profile` (Boolean) Auto created users will have access to their profile page and will be able to perform actions such as generating an API key. Default to `false`.
- `auto_create_user` (Boolean) When set, authenticated users are automatically created in Artifactory. When not set, for every request from an SSO user, the user is temporarily associated with default groups (if such groups are defined), and the permissions for these groups apply. Without automatic user creation, you must manually create the user inside Artifactory to manage user permissions not attached to their default groups. Default to `false`.
- `remote_user_request_variable` (String) The name of the HTTP request variable to use for extracting the user identity. Default to `REMOTE_USER`.
- `sync_ldap_groups` (Boolean) When set, the user will be associated with the groups returned in the LDAP login response. Note that the user's association with the returned groups is persistent if the `auto_create_user` is set. Default to `false`.

## Import

Import is supported using the following syntax:

```shell
terraform import platform_http_sso_settings.my-http-sso-settings my-http-sso-settings
```
