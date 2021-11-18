# Required for Terraform 0.13 and up (https://www.terraform.io/upgrade-guides/0-13.html)
terraform {
  required_providers {
    project = {
      source  = "registry.terraform.io/jfrog/project"
      version = "0.0.2"
    }
  }
}

resource "project" "myproject" {
  key = "myproj"
  display_name = "My Project"
  description = "My Project"
  admin_privileges {
    manage_members = true
    manage_resources = true
    index_resources = true
  }
  max_storage_in_gigabytes = 10
  block_deployments_on_limit = false
  email_notification = true
}
