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
  key = "my_project"
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
  // users = ["user1","user2"]
  // groups {
  //   name = "somegroup"
  //   roles = ["admin","reader"]
  // }
  // group {
  //   name = "somegroup1"
  //   roles = ["admin","reader"]
  // }
  // member {
  //   name = "christian"
  //   roles = ["admin"]
  // }
  // member {
  //   name = "karol"
  //   roles = ["admin"]
  // }
  // project_admins {
  //   user_names = ["christian"]
  //   roles = ["some_admin_role"]
  // }
  // build_repository = "mybuild"
  // repositories = ["myrepo", "other-repo"]
}
