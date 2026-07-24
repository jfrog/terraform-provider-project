---
page_title: "Migrating to the jfrog_ namespace"
subcategory: "Guides"
description: |-
  How to migrate from the deprecated un-namespaced resource names to the new jfrog_ namespaced resources.
---

# Migrating to the `jfrog_` namespace

Starting with v1.10.0, every resource in this provider is available under the
`jfrog_` namespace (for example `jfrog_project` instead of `project`). The old,
un-namespaced names still work but are **deprecated** and will be removed in the
next major release.

| Deprecated name                     | New name                                  |
|-------------------------------------|-------------------------------------------|
| `project`                           | `jfrog_project`                           |
| `project_environment`               | `jfrog_project_environment`               |
| `project_group`                     | `jfrog_project_group`                     |
| `project_repository`                | `jfrog_project_repository`                |
| `project_role`                      | `jfrog_project_role`                      |
| `project_share_repository`          | `jfrog_project_share_repository`          |
| `project_share_repository_with_all` | `jfrog_project_share_repository_with_all` |
| `project_user`                      | `jfrog_project_user`                      |

## 1. Update the provider local name

Terraform derives the provider from a resource's type prefix. The deprecated
names (`project`, `project_*`) resolve to the local name `project`, while the
new `jfrog_*` names resolve to the local name `jfrog`. Update your
`required_providers` block so the provider is configured under `jfrog`:

```terraform
terraform {
  required_providers {
    jfrog = {
      source  = "jfrog/project"
      version = "~> 1.10"
    }
  }
}

provider "jfrog" {
  url          = "https://myinstance.jfrog.io"
  access_token = var.access_token
}
```

## 2. Rename the resources and add `moved` blocks

Renaming a resource type is **not** an automatic operation in Terraform. Use a
`moved` block so existing state is migrated in place instead of the resource
being destroyed and recreated:

```terraform
resource "jfrog_project" "myproject" {
  key          = "myproj"
  display_name = "My Project"
  # ...
}

moved {
  from = project.myproject
  to   = jfrog_project.myproject
}
```

Apply the change and confirm the plan reports a move (not a destroy/create).
Once applied, the `moved` block can be removed.

~> **Warning:** If you rename a resource without a `moved` block (or without
running `terraform state mv`), Terraform will plan to **destroy** the old
resource and **create** a new one, which can be disruptive for resources such as
`jfrog_project`.

## 3. Remove the deprecated provider configuration

After all resources are migrated, remove the old `project` local name from
`required_providers` and delete any `provider "project"` blocks.
