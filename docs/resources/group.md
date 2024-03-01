---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "project_group Resource - terraform-provider-project"
subcategory: ""
description: |-
  Add a group as project member. Element has one to one mapping with the JFrog Project Groups API https://jfrog.com/help/r/jfrog-rest-apis/update-group-in-project. Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if admin_privileges.manage_resoures is enabled.
---

# project_group (Resource)

Add a group as project member. Element has one to one mapping with the [JFrog Project Groups API](https://jfrog.com/help/r/jfrog-rest-apis/update-group-in-project). Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.

## Example Usage

```terraform
resource "project_group" "mygroup" {
  project_key = "myproj"
  name        = "mygroup"
  roles       = ["Viewer"]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of an artifactory group.
- `project_key` (String) The key of the project to which the group should be assigned to.
- `roles` (Set of String) List of pre-defined Project or custom roles

### Read-Only

- `id` (String) The ID of this resource.

## Import

Import is supported using the following syntax:

```shell
terraform import project_group.mygroup project_key:groupname
```