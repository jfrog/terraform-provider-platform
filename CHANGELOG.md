## 1.9.0 (July 19, 2024). Tested on Artifactory 7.84.17 with Terraform 1.9.2 and OpenTofu 1.7.3

NOTES:

* resource/platform_myjfrog_ip_allowlist is being deprecated and is moved to new [MyJFrog provider](https://registry.terraform.io/providers/jfrog/myjfrog/latest). Use the `myjfrog_ip_allowlist` resource there instead. PR: [#105](https://github.com/jfrog/terraform-provider-platform/pull/105)

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
