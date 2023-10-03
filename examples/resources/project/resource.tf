resource "project" "myproject" {
  key = "myproj"
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
  use_project_role_resource  = true

  member {
    name  = "user1"
    roles = ["developer","project admin"]
  }

  member {
    name  = "user2"
    roles = ["developer"]
  }

  group {
    name = "dev-group"
    roles = ["developer"]
  }

  group {
    name = "release-group"
    roles = ["release manager"]
  }

  repos = ["docker-local", "rpm-local"]
}
