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

* `key` - (Required)
* `display_name` - (Required)
* `description` - (Optional)
* `admin_privileges` - (Required)
* `max_storage_in_gibabytes` - (Optional)
* `block_deployments_on_limit` - (Optional)
* `email_notification` - (Optional)
* `member` - (Optional)

### Member Argument

Member arguments has one to one mapping with the [JFrog Project Users API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateUserinProject). The following arguments are supported:

* `name` - (Required)
* `roles` - (Required)
