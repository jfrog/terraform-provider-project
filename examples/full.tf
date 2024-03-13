# Required for Terraform 0.13 and up (https://www.terraform.io/upgrade-guides/0-13.html)
terraform {
  required_providers {
    artifactory = {
      source  = "registry.terraform.io/jfrog/artifactory"
      version = "10.3.0"
    }
    project = {
      source  = "registry.terraform.io/jfrog/project"
      version = "1.5.0"
    }
  }
}

variable "artifactory_url" {
  type = string
  default = "http://localhost:8081"
}

provider "artifactory" {
  url = "${var.artifactory_url}"
}

provider "project" {
  url = "${var.artifactory_url}"
}

variable "qa_roles" {
  type = list(string)
  default = ["READ_REPOSITORY","READ_RELEASE_BUNDLE", "READ_BUILD", "READ_SOURCES_PIPELINE", "READ_INTEGRATIONS_PIPELINE", "READ_POOLS_PIPELINE", "TRIGGER_PIPELINE"]
}

variable "devop_roles" {
  type = list(string)
  default = ["READ_REPOSITORY", "ANNOTATE_REPOSITORY", "DEPLOY_CACHE_REPOSITORY", "DELETE_OVERWRITE_REPOSITORY", "TRIGGER_PIPELINE", "READ_INTEGRATIONS_PIPELINE", "READ_POOLS_PIPELINE", "MANAGE_INTEGRATIONS_PIPELINE", "MANAGE_SOURCES_PIPELINE", "MANAGE_POOLS_PIPELINE", "READ_BUILD", "ANNOTATE_BUILD", "DEPLOY_BUILD", "DELETE_BUILD",]
}

# Artifactory resources

resource "artifactory_user" "user1" {
  name     = "user1"
  email    = "test-user1@artifactory-terraform.com"
  groups   = ["readers"]
  password = "Passw0rd!"
}

resource "artifactory_user" "user2" {
  name     = "user2"
  email    = "test-user2@artifactory-terraform.com"
  groups   = ["readers"]
  password = "Passw0rd!"
}

resource "artifactory_group" "qa-group" {
  name             = "qa"
  description      = "QA group"
  admin_privileges = false
}

resource "artifactory_group" "release-group" {
  name             = "release"
  description      = "release group"
  admin_privileges = false
}

resource "artifactory_local_docker_v2_repository" "docker-local" {
  key             = "docker-local"
  description     = "hello docker-local"
  tag_retention   = 3
  max_unique_tags = 5
}

resource "artifactory_remote_npm_repository" "npm-remote" {
  key                                  = "npm-remote"
  url                                  = "https://registry.npmjs.org"
  mismatching_mime_types_override_list = "application/json,application/xml"
}

# Project resources

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

resource "project_group" "qa" {
  project_key = project.myproject.key
  name        = "qa"
  roles       = ["qa"]
}

resource "project_group" "release" {
  project_key = project.myproject.key
  name        = "release"
  roles       = ["release manager"]
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

resource "project_repository" "docker-local" {
  project_key = project.myproject.key
  key         = "docker-local"
}

resource "project_repository" "npm-local" {
  project_key = project.myproject.key
  key         = "npm-local"
}

resource "project_environment" "myenv" {
  project_key = project.myproj.key
  name        = "myenv"
}
