## 2.2.1 (January 8, 2025). Tested on Artifactory 7.98.13 with Terraform 1.10.3 and OpenTofu 1.8.8

BUG FIXES:

* resource/platform_reverse_proxy: Fix validation for optional attributes not working for variables. Issue: [#186](https://github.com/jfrog/terraform-provider-platform/issues/186) PR: [#188](https://github.com/jfrog/terraform-provider-platform/pull/188)

## 2.2.0 (January 3, 2025). Tested on Artifactory 7.98.12 with Terraform 1.10.3 and OpenTofu 1.8.8

FEATURES:

**New Resource:**

* `platform_group_members` - Resource to manage group membership. Issue: [#169](https://github.com/jfrog/terraform-provider-platform/issues/169) PR: [#187](https://github.com/jfrog/terraform-provider-platform/pull/187)

IMPROVEMENTS:

* resource/platform_group: Add `use_group_members_resource` attribute to control if the resource uses `members` attribute or not. Issue: [#169](https://github.com/jfrog/terraform-provider-platform/issues/169) PR: [#187](https://github.com/jfrog/terraform-provider-platform/pull/187)

## 2.1.0 (December 20, 2024). Tested on Artifactory 7.98.11 with Terraform 1.10.3 and OpenTofu 1.8.7

FEATURES:

**New Resource:**

* `platform_http_sso_settings` - Resource to manage HTTP SSO settings. PR: [#185](https://github.com/jfrog/terraform-provider-platform/pull/185)

## 2.0.0 (December 18, 2024). Tested on Artifactory 7.98.11 with Terraform 1.10.2 and OpenTofu 1.8.7

NOTES:

* provider: Deprecated attributes `check_license` and `myjfrog_api_token` are removed.
* resource/platform_myjfrog_ip_allowlist: Deprecated resource is removed.
* resource/platform_saml_settings: Deprecated attribute `no_auto_user_creation` is removed.

IMPROVEMENTS:

* resource/platform_oidc_configuration: Add `use_default_proxy` attribute to support the use of default proxy configuration.

PR: [#181](https://github.com/jfrog/terraform-provider-platform/pull/181)

## 1.19.1 (December 10, 2024). Tested on Artifactory 7.98.10 with Terraform 1.10.1 and OpenTofu 1.8.7

BUG FIXES:

* resource/platform_group: Fix "Provider produced inconsistent result after apply" error for attribute `members`. Issue: [#176](https://github.com/jfrog/terraform-provider-platform/issues/176) PR: [#177](https://github.com/jfrog/terraform-provider-platform/pull/177)

## 1.19.0 (December 3, 2024). Tested on Artifactory 7.98.9 with Terraform 1.10.0 and OpenTofu 1.8.6

FEATURES:

**New Resource:**

* `platform_crowd_settings` - Resource to manage Crowd/JIRA authentication provider. PR: [#167](https://github.com/jfrog/terraform-provider-platform/pull/167)

BUG FIXES:

* resource/platform_saml_settings: Fix `Value Conversion Error` for attribute `ldap_group_settings`. Issue: [#168](https://github.com/jfrog/terraform-provider-platform/issues/168) PR: [#171](https://github.com/jfrog/terraform-provider-platform/pull/171)

## 1.18.2 (November 27, 2024). Tested on Artifactory 7.98.9 with Terraform 1.9.8 and OpenTofu 1.8.6

BUG FIXES:

* resource/platform_permission: Fix permission resource not being deleted correctly. Permission resources that were in the Terraform state from previously apply but now are removed from the plan were not deleted. Now we check for the removal and delete the permission resource correctly. Issue: [#141](https://github.com/jfrog/terraform-provider-platform/issues/141) PR: [#165](https://github.com/jfrog/terraform-provider-platform/pull/165)

## 1.18.1 (November 26, 2024). Tested on Artifactory 7.98.9 with Terraform 1.9.8 and OpenTofu 1.8.6

BUG FIXES:

* resource/platform_oidc_configuration: Update validation for `issuer` attribute to support GitHub actions customization for enterprise. See [Customizing the issuer value for an enterprise](https://docs.github.com/en/enterprise-cloud@latest/actions/security-for-github-actions/security-hardening-your-deployments/about-security-hardening-with-openid-connect#customizing-the-issuer-value-for-an-enterprise). PR: [#163](https://github.com/jfrog/terraform-provider-platform/pull/163) and [#164](https://github.com/jfrog/terraform-provider-platform/pull/164)

## 1.18.0 (November 21, 2024). Tested on Artifactory 7.98.8 with Terraform 1.9.8 and OpenTofu 1.8.5

IMPROVEMENTS:

* resource/platform_saml_settings: Add `ldap_group_settings` attribute to support LDAP groups synchronization. Issue: [#154](https://github.com/jfrog/terraform-provider-platform/issues/154) PR: [#158](https://github.com/jfrog/terraform-provider-platform/pull/158)

## 1.17.0 (November 15, 2024). Tested on Artifactory 7.98.8 with Terraform 1.9.8 and OpenTofu 1.8.5

FEATURES:

**New Resource:**

* `platform_group` - Resource to manage Group, using [Platform API](https://jfrog.com/help/r/jfrog-rest-apis/groups). This replaces the `artifactory_group` resource in [Artifactory provider](https://registry.terraform.io/providers/jfrog/artifactory/latest/docs/resources/group), which uses the (deprecated) Artifactory Security API. PR: [#155](https://github.com/jfrog/terraform-provider-platform/pull/155)

## 1.16.0 (November 1, 2024). Tested on Artifactory 7.98.7 with Terraform 1.9.8 and OpenTofu 1.8.4

IMPROVEMENTS:

* resource/platform_oidc_identity_mapping: Add `username_pattern` and `groups_pattern` attributes to username and groups patterns. Attribute `scope` is now optional to support patterns. Issue: [#145](https://github.com/jfrog/terraform-provider-platform/issues/145) PR: [#147](https://github.com/jfrog/terraform-provider-platform/pull/147)

## 1.15.1 (October 18, 2024). Tested on Artifactory 7.90.14 with Terraform 1.9.8 and OpenTofu 1.8.3

IMPROVEMENTS:

* resource/artifactory_local_repository_multi_replication: Update validation for `actions.users` and `actions.groups` attributes to allow empty list. Issue: [#142](https://github.com/jfrog/terraform-provider-platform/issues/142) PR: [#143](https://github.com/jfrog/terraform-provider-platform/pull/143)

## 1.15.0 (October 16, 2024). Tested on Artifactory 7.90.14 with Terraform 1.9.7 and OpenTofu 1.8.3

IMPROVEMENTS:

* provider: Add `tfc_credential_tag_name` configuration attribute to support use of different/[multiple Workload Identity Token in Terraform Cloud Platform](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/manual-generation#generating-multiple-tokens). Issue: [#68](https://github.com/jfrog/terraform-provider-shared/issues/68) PR: [#139](https://github.com/jfrog/terraform-provider-platform/pull/139)

## 1.14.0 (October 11, 2024). Tested on Artifactory 7.90.14 with Terraform 1.9.7 and OpenTofu 1.8.3

IMPROVEMENTS:

* resource/platform_oidc_configuration: Add `project_key` attribute to support project scope OIDC configuration.
* resource/platform_oidc_identity_mapping: Add `project_key` attribute to support project scope OIDC identity mapping.
* resource/platform_oidc_identity_mapping: Add support for `applied-permissions/roles` to `scope` attribute to support project role scope.

PR: [#137](https://github.com/jfrog/terraform-provider-platform/pull/137) and [#138](https://github.com/jfrog/terraform-provider-platform/pull/138)

## 1.13.1 (October 8, 2024). Tested on Artifactory 7.90.13 with Terraform 1.9.7 and OpenTofu 1.8.3

BUG FIXES:

* resource/platform_saml_settings: Fix `certificate` attribute not being set as 'sensitive' so its value is redacted in CLI output/log. Issue: [#135](https://github.com/jfrog/terraform-provider-platform/issues/135) PR: [#136](https://github.com/jfrog/terraform-provider-platform/pull/136)

## 1.13.0 (October 4, 2024). Tested on Artifactory 7.90.13 with Terraform 1.9.7 and OpenTofu 1.8.3

NOTES:

* resource/platform_saml_settings: `no_auto_user_creation` attribute is being deprecated in favor of `auto_user_creation`.

BUG FIXES:

* resource/platform_saml_settings: Fix `no_auto_user_creation` attribute has no effect. Replace it with new attribute `auto_user_creation` to match SAML Settings in the UI and REST API. Issue: [#133](https://github.com/jfrog/terraform-provider-platform/issues/133) PR: [#134](https://github.com/jfrog/terraform-provider-platform/pull/134)

## 1.12.0 (September 16, 2024). Tested on Artifactory 7.90.10 with Terraform 1.9.5 and OpenTofu 1.8.2

FEATURES:

**New Resource:**
* `platform_aws_iam_role` - Resource to manage AWS IAM role. PR: [#125](https://github.com/jfrog/terraform-provider-platform/pull/125)

## 1.11.1 (September 9, 2024). Tested on Artifactory 7.90.9 with Terraform 1.9.5 and OpenTofu 1.8.2

IMPROVEMENTS:

* resource/platform_workers_service: Replace beta warning message in documentation with GA note. PR: [#124](https://github.com/jfrog/terraform-provider-platform/pull/124)

## 1.11.0 (August 12, 2024). Tested on Artifactory 7.90.7 with Terraform 1.9.4 and OpenTofu 1.8.1

FEATURES:

**New Resource:**
* `platform_saml_settings` - Resource to manage SAML SSO settings. PR: [#118](https://github.com/jfrog/terraform-provider-platform/pull/118)
* `platform_scim_user` - Resource to manage SCIM user. PR: [#120](https://github.com/jfrog/terraform-provider-platform/pull/120)
* `platform_scim_group` - Resource to manage SCIM group. PR: [#120](https://github.com/jfrog/terraform-provider-platform/pull/120)

## 1.10.0 (July 21, 2024). Tested on Artifactory 7.84.17 with Terraform 1.9.2 and OpenTofu 1.7.3

NOTES:

* provider: Attribute `myjfrog_api_token` is being deprecated. Issue: [#99](https://github.com/jfrog/terraform-provider-platform/issues/99) PR: [#114](https://github.com/jfrog/terraform-provider-platform/pull/114)
* resource/platform_myjfrog_ip_allowlist is being deprecated and is moved to new [MyJFrog provider](https://registry.terraform.io/providers/jfrog/myjfrog/latest). Use the `myjfrog_ip_allowlist` resource there instead. Issue: [#99](https://github.com/jfrog/terraform-provider-platform/issues/99 PR: [#113](https://github.com/jfrog/terraform-provider-platform/pull/113)

## 1.9.0 (July 19, 2024). Tested on Artifactory 7.84.17 with Terraform 1.9.2 and OpenTofu 1.7.3

IMPROVEMENTS:

* resource/platform_oidc_configuration: Add `Azure` option for `provider_type` attribute. PR: [#112](https://github.com/jfrog/terraform-provider-platform/pull/112)

## 1.8.2 (July 16, 2024). Tested on Artifactory 7.84.17 with Terraform 1.9.2 and OpenTofu 1.7.3

BUG FIXES:

* resource/platform_global_role, resource/platform_oidc_configuration, resource/platform_oidc_identity_mapping, resource/platform_permission, resource/platform_reverse_proxy, resource/platform_workers_service: Fix incorrect API error handling. Issue: [#104](https://github.com/jfrog/terraform-provider-platform/issues/104) PR: [#105](https://github.com/jfrog/terraform-provider-platform/pull/105)

## 1.8.1 (July 3, 2024). Tested on Artifactory 7.84.16 with Terraform 1.9.1 and OpenTofu 1.7.2

BUG FIXES:

* resource/platform_oidc_configuration: Fix `provider_type` attribute value stored incorrectly when resource is imported. Issue: [#102](https://github.com/jfrog/terraform-provider-platform/issues/102) PR: [#103](https://github.com/jfrog/terraform-provider-platform/pull/103)

## 1.8.0 (June 20, 2024). Tested on Artifactory 7.84.15 with Terraform 1.8.5 and OpenTofu 1.7.2

NOTES:

* provider: `check_license` attribute is deprecated and provider no longer checks Artifactory license during initialization. It will be removed in the next major version release.

BUG FIXES:

* provider: Fix incomplete provider initialization if Artifactory version check fails.

IMPROVEMENTS:

* provider: Now allows JFrog Access Token to be unset (i.e. MyJFrog API token is set and only `platform_myjfrog_ip_allowlist` resource is used). Warning message is displayed when either token is not set.

Issue: [#87](https://github.com/jfrog/terraform-provider-platform/issues/87) PR: [#97](https://github.com/jfrog/terraform-provider-platform/pull/97)

## 1.7.4 (May 8, 2024). Tested on Artifactory 7.84.14 with Terraform 1.8.4 and OpenTofu 1.7.2

BUG FIXES:

* resource/platform_permission: Fix state upgrader crash when upgrading from 1.7.2 to 1.7.3 with no `groups` or `users` attribute set. Issue: [#75](https://github.com/jfrog/terraform-provider-platform/issues/75) PR: [#76](https://github.com/jfrog/terraform-provider-platform/pull/76)

## 1.7.3 (May 7, 2024)

BUG FIXES:

* resource/platform_oidc_configuration: Remove trailing slash from GitHub provider URL validation. Issue: [#71](https://github.com/jfrog/terraform-provider-platform/issues/71) PR: [#72](https://github.com/jfrog/terraform-provider-platform/pull/72)
* resource/platform_permission: Fix state drift when `*.actions.users` or `*.actions.groups` are set to empty set (`[]`). These 2 attributes now must either be `null`/not set, or a set of at least 1 item. Existing Terraform state with `[]` should be migrated to `null` automatically by provider. Issue: [#70](https://github.com/jfrog/terraform-provider-platform/issues/70) PR: [#73](https://github.com/jfrog/terraform-provider-platform/pull/73)

## 1.7.2 (May 1, 2024)

BUG FIXES:

* resource/platform_permission: Make `name` attribute trigger resource replacement if changed. Issue: [#64](https://github.com/jfrog/terraform-provider-platform/issues/64) PR: [#66](https://github.com/jfrog/terraform-provider-platform/pull/66)

## 1.7.1 (Apr 15, 2024)

BUG FIXES:

* provider: Fix crashes when `url` attribute is used to set JFrog platform URL (vs env var). Issue: [#57](https://github.com/jfrog/terraform-provider-platform/issues/57) PR: [#58](https://github.com/jfrog/terraform-provider-platform/pull/58)

## 1.7.0 (Apr 12, 2024)

FEATURES:

* provider: Add support for Terraform Cloud Workload Identity Token. Issue: [#30](https://github.com/jfrog/terraform-provider-platform/issues/30) PR: [#54](https://github.com/jfrog/terraform-provider-platform/pull/54)

## 1.6.0 (Apr 5, 2024)

FEATURES:

* **New Resource:** `platform_myjfrog_ip_allowlist`: Resource to manage MyJFrog IP allowlist. PR: [#50](https://github.com/jfrog/terraform-provider-platform/pull/50) Issue: [#27](https://github.com/jfrog/terraform-provider-platform/issues/27)

## 1.5.1 (Apr 3, 2024)

BUG FIXES:

* resource/platform_permission: Update documentation for target `name` attribute for `ANY` repository types. Correct values should be `ANY LOCAL`, `ANY REMOTE`, or `ANY DISTRIBUTION`. Note the removal of underscore character. Issue: [#48](https://github.com/jfrog/terraform-provider-platform/issues/48) PR: [#49](https://github.com/jfrog/terraform-provider-platform/pull/49)

## 1.5.0 (Mar 26, 2024)

FEATURES:

* **New Resource:** `platform_oidc_configuration` and `platform_oidc_identity_mapping`: PR: [#47](https://github.com/jfrog/terraform-provider-platform/pull/47) Issue: [#26](https://github.com/jfrog/terraform-provider-platform/issues/26), [#29](https://github.com/jfrog/terraform-provider-platform/issues/29), [#31](https://github.com/jfrog/terraform-provider-platform/issues/31), [#38](https://github.com/jfrog/terraform-provider-platform/issues/38)

## 1.4.1 (Mar 18, 2024)

BUG FIXES:

* Fix HTTP response error handling due to change of behavior (for better consistency) from Resty client. PR: [#42](https://github.com/jfrog/terraform-provider-platform/pull/42)

## 1.4.0 (Mar 6, 2024)

FEATURES:

* **New Resource:** `platform_global_role`: PR: [#35](https://github.com/jfrog/terraform-provider-platform/pull/35)

## 1.3.0 (Feb 29, 2024)

FEATURES:

* **New Resource:** `platform_permission`: PR: [#33](https://github.com/jfrog/terraform-provider-platform/pull/33) Issue: [#32](https://github.com/jfrog/terraform-provider-platform/issues/32)

## 1.2.0 (Jan 4, 2024). Tested on Artifactory 7.71.11 with Terraform CLI v1.6.6

FEATURES:

* **New Resource:** `platform_license`: PR: [#20](https://github.com/jfrog/terraform-provider-platform/pull/20) Issue: [#12](https://github.com/jfrog/terraform-provider-platform/issues/12)

## 1.1.0 (Dec 14, 2023). Tested on Artifactory 7.71.10 with Terraform CLI v1.6.6

FEATURES:

* **New Resource:** `platform_reverse_proxy`: PR: [#13](https://github.com/jfrog/terraform-provider-platform/pull/13) Issue: [#11](https://github.com/jfrog/terraform-provider-platform/issues/11)

## 1.0.1 (Nov 28, 2023). Tested on Artifactory 7.71.5 with Terraform CLI v1.6.4

IMPROVEMENTS:

* Bump github.com/hashicorp/terraform-plugin-go from 0.19.0 to 0.19.1: PR: [6](https://github.com/jfrog/terraform-provider-platform/pull/6)
* Bump github.com/hashicorp/terraform-plugin-testing from 1.5.0 to 1.5.1: PR: [5](https://github.com/jfrog/terraform-provider-platform/pull/5)
* Bump github.com/go-resty/resty/v2 from 2.7.0 to 2.10.0: PR: [4](https://github.com/jfrog/terraform-provider-platform/pull/4)

## 1.0.0 (Nov 27, 2023)

FEATURES:

* **New Resource:** `platform_workers_service`: PR: [#2](https://github.com/jfrog/terraform-provider-platform/pull/2)
