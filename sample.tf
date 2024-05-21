terraform {
  required_providers {
    project = {
      source  = "jfrog/project"
      version = "1.5.3"
    }
  }
}

variable "qa_roles" {
  type    = list(string)
  default = ["READ_REPOSITORY", "READ_RELEASE_BUNDLE", "READ_BUILD", "READ_SOURCES_PIPELINE", "READ_INTEGRATIONS_PIPELINE", "READ_POOLS_PIPELINE", "TRIGGER_PIPELINE"]
}

variable "devop_roles" {
  type    = list(string)
  default = ["READ_REPOSITORY", "ANNOTATE_REPOSITORY", "DEPLOY_CACHE_REPOSITORY", "DELETE_OVERWRITE_REPOSITORY", "TRIGGER_PIPELINE", "READ_INTEGRATIONS_PIPELINE", "READ_POOLS_PIPELINE", "MANAGE_INTEGRATIONS_PIPELINE", "MANAGE_SOURCES_PIPELINE", "MANAGE_POOLS_PIPELINE", "READ_BUILD", "ANNOTATE_BUILD", "DEPLOY_BUILD", "DELETE_BUILD", ]
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

resource "project_user" "user1" {
  project_key = project.myproject.key
  name        = "user1"
  roles       = ["developer","project admin"]
}

resource "project_user" "user2" {
  project_key = project.myproject.key
  name        = "user2"
  roles       = ["developer"]
}

resource "project_group" "dev-group" {
  project_key = project.myproject.key
  name        = "dev-group"
  roles       = ["developer"]
}

resource "project_group" "release-group" {
  project_key = project.myproject.key
  name        = "release-group"
  roles       = ["release manager"]
}

resource "project_repository" "docker-local" {
  project_key = project.myproject.key
  key         = "docker-local"
}

resource "project_repository" "rpm-local" {
  project_key = project.myproject.key
  key         = "rpm-local"
}

resource "project_environment" "myenv" {
  project_key = project.myproj.key
  name        = "myenv"
}

resource "project_role" "qa" {
  project_key  = project.myproject.key
  name         = "qa"
  type         = "CUSTOM"
  environments = ["DEV"]
  actions      = var.qa_roles
}

resource "project_role" "devop" {
  project_key  = project.myproject.key
  name         = "devop"
  type         = "CUSTOM"
  environments = ["DEV", "PROD"]
  actions      = var.devop_roles
}
