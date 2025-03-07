---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "project_share_repository_with_all Resource - terraform-provider-project"
subcategory: ""
description: |-
  Share a local or remote repository with all projects. Project Members of the target project are granted actions to the shared repository according to their Roles and Role actions assigned in the target Project. Requires a user assigned with the 'Administer the Platform' role.
  ->Only available for Artifactory 7.90.1 or later.
---

# project_share_repository_with_all (Resource)

Share a local or remote repository with all projects. Project Members of the target project are granted actions to the shared repository according to their Roles and Role actions assigned in the target Project. Requires a user assigned with the 'Administer the Platform' role.

->Only available for Artifactory 7.90.1 or later.

## Example Usage

```terraform
resource "project_share_repository_with_all" "myprojectsharerepo" {
  repo_key = "myrepo-generic-local"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `repo_key` (String) The key of the repository.

### Optional

- `read_only` (Boolean) Share repository with all Projects in Read-Only mode to avoid any changes or modifications of the shared content.

->Only available for Artifactory 7.94.0 or later.

## Import

Import is supported using the following syntax:

```shell
terraform import project_share_repository_with_all.myprojectsharerepo repo_key
```
