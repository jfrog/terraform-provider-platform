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