---
page_title: "Migrating from the `project` resource"
---

Starting with provider version 1.9.9, the project resource is available under the namespaced name `project_project`. The original `project` name is deprecated and will be removed in the next major version.

## Why the resource was renamed

`project` was the only resource in this provider that was not prefixed with the provider name. That made configurations ambiguous to read, since `resource "project" "myproject"` gives no indication that the resource belongs to the JFrog project domain, and it was inconsistent with every other resource in the provider (`project_group`, `project_repository`, `project_role`, and so on). The non-namespaced name also prevented tooling that relies on the `<provider>_<resource>` convention, such as Crossplane Upjet, from consuming the provider.

## Nothing breaks if you do nothing

The `project` resource continues to work exactly as before. It shares its schema, CRUD behavior, state upgraders, and import support with `project_project` — the two are the same implementation under two names. The only difference is that `project` now emits a deprecation warning on every `terraform plan` and `terraform apply`:

```
Warning: Deprecated Resource

The `project` resource is deprecated and will be removed in the next major
version. Use the namespaced `project_project` resource instead.
```

You should migrate before the next major release, but you can do it on your own schedule.

## Migrating with a `moved` block

Rename the resource in your configuration and add a [`moved` block](https://developer.hashicorp.com/terraform/language/moved) telling Terraform where the state should go. This is a state-only operation: your project in Artifactory is never destroyed or recreated.

Before:

```hcl
resource "project" "my_project" {
  key          = "myproj"
  display_name = "My Project"
  description  = "My Project"

  admin_privileges {
    manage_members           = true
    manage_resources         = true
    manage_remote_repository = true
    index_resources          = true
  }

  max_storage_in_gibibytes   = 10
  block_deployments_on_limit = false
  email_notification         = true
}
```

After:

```hcl
resource "project_project" "my_project" {
  key          = "myproj"
  display_name = "My Project"
  description  = "My Project"

  admin_privileges {
    manage_members           = true
    manage_resources         = true
    manage_remote_repository = true
    index_resources          = true
  }

  max_storage_in_gibibytes   = 10
  block_deployments_on_limit = false
  email_notification         = true
}

moved {
  from = project.my_project
  to   = project_project.my_project
}
```

Remember to update any references to the resource elsewhere in your configuration, for example `project_key = project.my_project.key` becomes `project_key = project_project.my_project.key`.

Running `terraform plan` reports the move and no infrastructure changes:

```
# project.my_project has moved to project_project.my_project

Plan: 0 to add, 0 to change, 0 to destroy.
```

Apply it, and then remove the `moved` block once every workspace that shares this configuration has been applied. A follow-up plan reports `No changes` with no deprecation warnings.

## `terraform state mv` does not work here

The Terraform CLI refuses to move state between two different resource types:

```
Error: Invalid state move request

Cannot move project.my_project to project_project.my_project: resource types
don't match.
```

The `moved` block is the supported migration path.
