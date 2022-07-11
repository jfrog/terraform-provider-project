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
