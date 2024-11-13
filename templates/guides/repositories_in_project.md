---
page_title: "Adding repositories to the project"
---

The guide provides information and the example on how to add repositories to the project.

## Resources creation sequence

1. Create `project` resource
2. Create repository resource(s), the ordering of these first 2 steps doesn't matter.
3. Create `project_repository` resource, using attributes from #1 and #2 as reference values for this resource

## Artifactory repository state drift

When a repository in Artifactory is assigned to a project, the API field `projectKey` is set with the project's key. While using the `project_key` attribute in the Artifactory provider to set the project key for the repository is possible, we **strongly** recommend using the `project_repository` resource instead.

However the next time `terraform plan` or `terraform apply` is run, a state drift will occur for the `project_key` attribute. To avoid this, use Terraform meta argument `lifecycle.ignore_changes`. e.g.

```hcl
resource "artifactory_local_docker_v2_repository" "docker-v2-local" {
  key                   = "myproj-docker-v2-local"
  tag_retention         = 3
  max_unique_tags       = 5
  project_environments  = ["PROD"]

  lifecycle {
    ignore_changes = [
      project_key
    ]
  }
}
```

## Full HCL example

```hcl
terraform {
  required_providers {
    artifactory = {
      source  = "jfrog/artifactory"
      version = "11.5.0"
    }
    project = {
      source  = "jfrog/project"
      version = "1.7.1"
    }
  }
}

provider "artifactory" {
  // supply JFROG_ACCESS_TOKEN / JFROG_URL as env vars
}

provider "project" {
  // supply JFROG_ACCESS_TOKEN / JFROG_URL as env vars
}

resource "project" "myproject" {
  key          = "myproj"
  display_name = "My Project"
  description  = "My Project"
  admin_privileges {
    manage_members   = true
    manage_resources = true
    index_resources  = true
  }
  max_storage_in_gibibytes   = 10
  block_deployments_on_limit = false
  email_notification         = true
}

resource "artifactory_local_docker_v2_repository" "docker-v2-local" {
  key                   = "docker-v2-local"
  tag_retention         = 3
  max_unique_tags       = 5
  project_environments  = ["PROD"]

  lifecycle {
    ignore_changes = [
      project_key
    ]
  }
}

resource "project_repository" "myproject-docker-v2-local" {
  project_key = project.myproject.key
  key         = artifactory_local_docker_v2_repository.docker-v2-local.key
} 
```
