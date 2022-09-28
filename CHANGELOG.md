## 1.1.8 (Sep 28, 2022). Tested on Artifactory 7.41.13 and Xray 3.57.6

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
