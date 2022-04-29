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
  
