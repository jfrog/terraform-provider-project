# Required for Terraform 0.13 and up (https://www.terraform.io/upgrade-guides/0-13.html)
terraform {
  required_providers {
    projects = {
      source  = "registry.terraform.io/jfrog/projects"
      version = "0.0.1"
    }
  }
}
resource "jfrog_projects" "myproject" {

  repositories = ["myrepo", "other-repo"]
  users = ["user1","user2"]
  groups {
    name = "somegroup"
    roles = ["admin","reader"]
  }
  group {
    name = "somegroup1"
    roles = ["admin","reader"]
  }
  member {
    name = "christian"
    roles = ["admin"]
  }
  member {
    name = "karol"
    roles = ["admin"]
  }
  build_repository = "mybuild"
  quota  {
    max_storage = "10gb"
    block_deployments_on_limit = true
  }
  project_admins {
    user_names = ["christian"]
    roles = ["some_admin_role"]
  }
  admin_privs {
    manage_resources = true
    index_resources = true
    manage_members = true
  }
}

