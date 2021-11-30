# Artifactory Project Resource

Provides an Artifactory project resource. This can be used to create and manage Artifactory project, maintain users/groups/roles/repos.

## Example Usage

```hcl
# Create a new Artifactory project called my-project
resource "project" "my-project" {
  key                      = "myprj"
  display_name             = "My Project"
  max_storage_in_gibabytes = 10

  admin_privileges {
    manage_members   = true
    manage_resources = true
    index_resources  = true
  }

  member {
    name  = "user1"
    roles = ["developer","project admin"]
  }
}
```

## Argument Reference

Project argument have an almost one to one mapping with the [JFrog Projects API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-PROJECTS). The following arguments are supported:

* `key` - (Required) The Project Key is added as a prefix to resources created within a Project. This field is mandatory and supports only 3 - 6 lowercase alphanumeric characters. Must begin with a letter. For example: us1a.
* `display_name` - (Required) Also known as project name on the UI
* `description` - (Optional)
* `admin_privileges` - (Required)
* `max_storage_in_gibabytes` - (Optional) Storage quota in GB. Must be 1 or larger
* `block_deployments_on_limit` - (Optional)
* `email_notification` - (Optional) Alerts will be sent when reaching 75% and 95% of the storage quota. Serves as a notification only and is not a blocker
* `member` - (Optional) Member of the project. Must be existing Artifactory user.

### Member Argument

Member arguments has one to one mapping with the [JFrog Project Users API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateUserinProject). The following arguments are supported:

* `name` - (Required)
* `roles` - (Required)
