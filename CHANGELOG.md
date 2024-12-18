## 1.9.3 (December 19, 2024). Tested on Artifactory 7.98.11 with Terraform 1.10.3 and OpenTofu 1.8.7

BUG FIXES:

* resource/project_environment: Fix error handling during creation/update where state shouldn't be stored/updated when there is API error. Issue: [#184](https://github.com/jfrog/terraform-provider-project/issues/184) PR: [#188](https://github.com/jfrog/terraform-provider-project/pull/188)

## 1.9.2 (November 26, 2024). Tested on Artifactory 7.98.9 with Terraform 1.9.8 and OpenTofu 1.8.6

IMPROVEMENTS:

* provider: Update to latest `github.com/hashicorp/terraform-plugin-docs` module to restore auto-generation of documentation. PR: [#183](https://github.com/jfrog/terraform-provider-project/pull/183)

## 1.9.1 (November 20, 2024). Tested on Artifactory 7.98.8 with Terraform 1.9.8 and OpenTofu 1.8.5

BUG FIXES:

* resource/project_user: Recreate resource correctly if `ignore_missing_user` set to `true`. Issue: [#175](https://github.com/jfrog/terraform-provider-project/issues/175) PR: [#177](https://github.com/jfrog/terraform-provider-project/pull/177)

## 1.9.0 (October 17, 2024). Tested on Artifactory 7.90.14 with Terraform 1.9.7 and OpenTofu 1.8.3

IMPROVEMENTS:

* provider: Add `tfc_credential_tag_name` configuration attribute to support use of different/[multiple Workload Identity Token in Terraform Cloud Platform](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/manual-generation#generating-multiple-tokens). Issue: [#68](https://github.com/jfrog/terraform-provider-shared/issues/68) PR: [#165](https://github.com/jfrog/terraform-provider-project/pull/165)

## 1.8.0 (September 17, 2024)

IMPROVEMENTS:

* resource/project_share_repository, resource/project_share_repository_with_all: Add `read_only` attribute to support sharing repository with project(s) in read-only mode. Issue: [#156](https://github.com/jfrog/terraform-provider-project/issues/156) PR: [#158](https://github.com/jfrog/terraform-provider-project/pull/158)

## 1.7.2 (August 8, 2024). Tested on Artifactory 7.90.6 with Terraform 1.9.3 and OpenTofu 1.8.1

IMPROVEMENTS:

* resource/project: Update documentation for `repos` attribute and "Adding repositories to the project" guide to reflect new way of assigning repository to project. PR: [#152](https://github.com/jfrog/terraform-provider-project/pull/152)

## 1.7.1 (July 25, 2024)

BUG FIXES:

* Fix Terraform manifest file is missing from checksum file. PR: [#148](https://github.com/jfrog/terraform-provider-project/pull/148)

## 1.7.0 (July 24, 2024). Tested on Artifactory 7.84.17 with Terraform 1.9.2 and OpenTofu 1.7.3

NOTES:

* provider: `check_license` attribute is deprecated and provider no longer checks Artifactory license during initialization. It will be removed in the next major version release.

FEATURES:

* **New Resource:** `project_share_repository` - New resource to share repository with a project.
* **New Resource:** `project_share_repository_with_all` - New resource to share repository with all projects.

Issue: [#80](https://github.com/jfrog/terraform-provider-project/issues/80)
PR: [#147](https://github.com/jfrog/terraform-provider-project/pull/147)

## 1.6.2 (June 4, 2024). Tested on Artifactory 7.84.14 with Terraform  and OpenTofu 1.7.2

IMPROVEMENTS:

* resource/project_group is migrated to Plugin Framework. PR: [#131](https://github.com/jfrog/terraform-provider-project/pull/131)
* resource/project_user, resource/project_role, and resource/project_repository are migrated to Plugin Framework. PR: [#133](https://github.com/jfrog/terraform-provider-project/pull/133)

## 1.6.1 (May 31, 2024)

BUG FIXES:

* Add validation to `roles` attribute for `project_group` and `project_user` resources to ensure at least one role is defined. Issue: [#126](https://github.com/jfrog/terraform-provider-project/issues/126) PR: [#129](https://github.com/jfrog/terraform-provider-project/pull/129)

IMPROVEMENTS:

* resource/project_environment is migrated to Plugin Framework. PR: [#130](https://github.com/jfrog/terraform-provider-project/pull/130)

## 1.6.0 (May 22, 2024)

FEATURES:

* provider: Add support for Terraform Cloud Workload Identity Token. PR: [#124](https://github.com/jfrog/terraform-provider-project/pull/124)

## 1.5.3 (May 13, 2024)

IMPROVEMENTS:

* resource/project is migrated to Plugin Framework. PR: [#120](https://github.com/jfrog/terraform-provider-project/pull/120)

## 1.5.2 (March 28, 2024)

IMPROVEMENTS:

* resource/project_repository: Add retry logic after resource creation to allow time for repository project assignment to be synced up. Issue: [#110](https://github.com/jfrog/terraform-provider-project/issues/110) PR: [#111](https://github.com/jfrog/terraform-provider-project/pull/111)

## 1.5.1 (March 14, 2024)

BUG FIXES:

* Fix HTTP response error handling due to change of behavior (for better consistency) from Resty client. PR: [#106](https://github.com/jfrog/terraform-provider-project/pull/106)

## 1.5.0 (March 13, 2024)

FEATURES:

* **New Resource:** `project_repository` - Separate resource to manage project repositories.
* resource/project: Add `use_project_repository_resource` attribute to toggle if `project` resource should use its `repos` attribute or not to manage project repositories. Should be set to `false` to continue using existing `repos` attribute.

IMPROVEMENTS:

* resource/project, resource/project_environment, resource/project_role: Fix documentation for `project_key` attribute to match validation. Issue: [#103](https://github.com/jfrog/terraform-provider-project/issues/103)

PR: [#105](https://github.com/jfrog/terraform-provider-project/pull/105)

## 1.4.0 (March 4, 2024)

FEATURES:

* **New Resource:** `project_user` - Separate resource to manage project memberships for users.
* **New Resource:** `project_group` - Separate resource to project project memberships for groups.
* resource/project: Add `use_project_user_resource` attribute to toggle if `project` resource should use its `member` or not to manage project users. Should be set to `false` to continue using existing `member` attribute.
* resource/project: Add `use_project_group_resource` attribute to toggle if `project` resource should use its `group` or not to manage project users. Should be set to `false` to continue using existing `group` attribute.
* resource/project: Switch default value for `use_project_role_resource` attribute to `true` so new provider practioners won't need to explicity set this attribute to use `project_role` resource.

Issue: [#83](https://github.com/jfrog/terraform-provider-project/issues/83), [#98](https://github.com/jfrog/terraform-provider-project/issues/98)

PR: [#101](https://github.com/jfrog/terraform-provider-project/pull/101)

## 1.3.5 (Feburary 9, 2024)

BUG FIXES:

* resource/project: Add retry logic for resource deletion. Issue: [#96](https://github.com/jfrog/terraform-provider-project/issues/96) PR: [#97](https://github.com/jfrog/terraform-provider-project/pull/97)

## 1.3.4 (October 31, 2023). Tested on Artifactory 7.71.8 and Xray 3.86.9

BUG FIXES:

* resource/project_environment: Fix incorrect environment from Artifactroy being matched and triggers a state drift. Issue: [#90](https://github.com/jfrog/terraform-provider-project/issues/90) PR: [#92](https://github.com/jfrog/terraform-provider-project/pull/92)

## 1.3.3 (October 18, 2023)

IMPROVEMENTS:

* resource/project_environment: Improve import documentation example
* resource/project_role: Add import documentation

PR: [#89](https://github.com/jfrog/terraform-provider-project/pull/89)

## 1.3.2 (October 12, 2023). Tested on Artifactory 7.68.14 and Xray 3.83.8

SECURITY:

* provider:
  * Bump golang.org/x/net from 0.11.0 to 0.17.0 PR: [#88](https://github.com/jfrog/terraform-provider-project/pull/88)

## 1.3.1 (October 4, 2023). Tested on Artifactory 7.68.13 and Xray 3.82.11

IMPROVEMENTS:

* resource/project:
  * Add clarification to `block_deployments_on_limit` attribute documentation with regards to difference behavior between self-hosted and cloud environments.
  * Update documentation and remove `role` attributes in HCL example.
* resource/project_role: Add HCL example to documentation.
* Update `sample.tf` to use `project_role` resource instead of `project.role` attribute.

PR: [#87](https://github.com/jfrog/terraform-provider-project/pull/87)

## 1.3.0 (September 25, 2023). Tested on Artifactory 7.68.11 and Xray 3.82.11

FEATURES:

* **New Resource:** `project_role` - Separate resource to manage project role.
* resource/project: Add `use_project_role_resource` attribute to toggle if `project` resource should use its `roles` or not to manage project roles. Should be set to `true` when using in conjunction with `project_role` resource.

Issue: [#85](https://github.com/jfrog/terraform-provider-project/issues/85)
PR: [#86](https://github.com/jfrog/terraform-provider-project/pull/86)

## 1.2.1 (August 30, 2023). Tested on Artifactory 7.63.14 and Xray 3.80.9

IMPROVEMENTS:

* resource/project: Update `max_storage_in_gibibytes` attribute to work with int64 type explicitly. Update validation to include max range. Issue: [#79](https://github.com/jfrog/terraform-provider-project/issues/79) PR: [#82](https://github.com/jfrog/terraform-provider-project/pull/82)

## 1.2.0 (August 29, 2023). Tested on Artifactory 7.63.14 and Xray 3.80.9

FEATURES:

* **New Resource:** `project_environment` Issue: [#78](https://github.com/jfrog/terraform-provider-project/issues/78) PR: [#81](https://github.com/jfrog/terraform-provider-project/pull/81)

## 1.1.16 (March 29, 2023). Tested on Artifactory 7.55.9 and Xray 3.69.3

IMPROVEMENTS:

* resource/project: Update `key` attribute validation to match Artifactory Project. PR: [#73](https://github.com/jfrog/terraform-provider-project/pull/73)

## 1.1.15 (February 24, 2023). Tested on Artifactory 7.49.8 and Xray 3.67.9

BUG FIXES:

* resource/project: Update `key` attribute validation to match Artifactory Project. PR: [#71](https://github.com/jfrog/terraform-provider-project/pull/71)

## 1.1.14 (February 23, 2023)

SECURITY:

* provider:
  * Bump golang.org/x/text from 0.3.7 to 0.3.8 PR: [#68](https://github.com/jfrog/terraform-provider-project/pull/68)
  * Bump golang.org/x/net from 0.0.0-20211029224645-99673261e6eb to 0.7.0 PR: [#69](https://github.com/jfrog/terraform-provider-project/pull/69)
  * Bump golang.org/x/crypto from 0.0.0-20210921155107-089bfa567519 to 0.1.0 PR: [#70](https://github.com/jfrog/terraform-provider-project/pull/70)

## 1.1.13 (Jan 5, 2023). Tested on Artifactory 7.49.8 and Xray 3.67.9

BUG FIXES:

* resource/project: Fix `block_deployments_on_limit` attribute value opposite to Artifactory Project web UI. Issue: [#62](https://github.com/jfrog/terraform-provider-project/issues/62) PR: [#67](https://github.com/jfrog/terraform-provider-project/pull/67)

## 1.1.12 (Dec 22, 2022). Tested on Artifactory 7.47.14 and Xray 3.62.4

BUG FIXES:

* resource/project: Update `key` attribute validation to match Artifactory Project. PR: [#66](https://github.com/jfrog/terraform-provider-project/pull/66)

## 1.1.11 (Dec 13, 2022).

IMPROVEMENTS:

* resource/project: added a guide on adding repositories to the project. PR: [#63](https://github.com/jfrog/terraform-provider-project/pull/63)

## 1.1.10 (Nov 18, 2022). Tested on Artifactory 7.46.11 and Xray 3.60.2

IMPROVEMENTS:

* resource/project: Additional note for repository configuration. Issue: [#58](https://github.com/jfrog/terraform-provider-project/issues/58) PR: [#61](https://github.com/jfrog/terraform-provider-project/pull/61)

## 1.1.9 (Nov 18, 2022). Tested on Artifactory 7.46.11 and Xray 3.60.2

IMPROVEMENTS:

* resource/project: Add recommendation note for attribute `repos`. Issue: [#58](https://github.com/jfrog/terraform-provider-project/issues/58) PR: [#60](https://github.com/jfrog/terraform-provider-project/pull/60)


## 1.1.8 (Sep 28, 2022). Tested on Artifactory 7.46.10 and Xray 3.59.4

BUG FIXES:

* resource/project: Ignore unassigning (non-existent) repository error when destroying project resource. PR: [#57](https://github.com/jfrog/terraform-provider-project/pull/57)

## 1.1.7 (Sep 8, 2022). Tested on Artifactory 7.41.12 and Xray 3.55.2

BUG FIXES:

* Remove parallel requests when adding/removing repos, users, groups, and roles with project. PR: [#54](https://github.com/jfrog/terraform-provider-project/pull/54)

## 1.1.6 (Aug 9, 2022). Tested on Artifactory 7.41.7 and Xray 3.54.5

BUG FIXES:

* resource/project: Update `key` attribute to support hyphen character. PR: [#53](https://github.com/jfrog/terraform-provider-project/pull/53)

## 1.1.5 (Aug 1, 2022)

BUG FIXES:

* provider: Fix license check to include license type. PR: [#52](https://github.com/jfrog/terraform-provider-project/pull/52)
* Update package `github.com/Masterminds/goutils` to 1.1.1 for [Dependeabot alert](https://github.com/jfrog/terraform-provider-project/security/dependabot/2)

## 1.1.4 (July 11, 2022). Tested on Artifactory 7.39.4 and Xray 3.51.3

BUG FIXES:

* updated to latest shared provider (internal ticket)
* update makefile to be consistent with other providers. Still doesn't do version substitution correctly PR: [#47](https://github.com/jfrog/terraform-provider-project/pull/47)

## 1.1.3 (July 1, 2022). Tested on Artifactory 7.39.4 and Xray 3.51.3

BUG FIXES:

* provider: Fix hardcoded HTTP user-agent string. PR: [#46](https://github.com/jfrog/terraform-provider-project/pull/46/)

## 1.1.2 (June 21, 2022). Tested on Artifactory 7.39.4 and Xray 3.51.3

IMPROVEMENTS:

* Bump shared module version

## 1.1.1 (May 27, 2022). Tested on Artifactory 7.38.10 and Xray 3.49.0

IMPROVEMENTS:

* Upgrade `gopkg.in/yaml.v3` to v3.0.0 for [CVE-2022-28948](https://nvd.nist.gov/vuln/detail/CVE-2022-28948) [[GH-42]](https://github.com/jfrog/terraform-provider-project/pull/42)

## 1.1.0 (Apr 13, 2022). Tested on Artifactory 7.37.15 and Xray 3.47.3

IMPROVEMENTS:

* Documentation improved for `project` resource to include limitations in Note section
[(#27)](https://github.com/jfrog/terraform-provider-project/issues/27),
[[GH-32]](https://github.com/jfrog/terraform-provider-project/pull/32)

## 1.0.4 (Mar 22, 2022)

BUG FIXES:

* Project Key validation fixed as per the latest version
[(#22)](https://github.com/jfrog/terraform-provider-project/issues/22),
[[GH-25]](https://github.com/jfrog/terraform-provider-project/pull/25)

## 1.0.3 (Feb 24, 2022)

BUG FIXES:

* resource/project: Add `ForceNew` field to `key` attribute to ensure resource is destroyed and recreated when key is changed
[(#23)](https://github.com/jfrog/terraform-provider-project/issues/23),
[[GH-24]](https://github.com/jfrog/terraform-provider-project/pull/24)
