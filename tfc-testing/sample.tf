terraform {
  cloud {
    organization = "jfrog-partnership-engineering"
    workspaces {
      name = "alexh"
    }
  }

  required_providers {
    project = {
      source  = "jfrog/project"
      version = "1.5.3"
    }
  }
}

provider "project" {
  url = "https://partnership.jfrog.io"
  oidc_provider_name = "terraform-cloud"
}

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
}
