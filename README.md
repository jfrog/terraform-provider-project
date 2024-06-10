[![Terraform & OpenTofu Acceptance Tests](https://github.com/jfrog/terraform-provider-project/actions/workflows/acceptance-tests.yml/badge.svg)](https://github.com/jfrog/terraform-provider-project/actions/workflows/acceptance-tests.yml)

# Terraform Provider for Artifactory Project

[![Actions Status](https://github.com/jfrog/terraform-provider-project/workflows/release/badge.svg)](https://github.com/jfrog/terraform-provider-project/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/jfrog/terraform-provider-project)](https://goreportcard.com/report/github.com/jfrog/terraform-provider-project)

## Quick Start

Create a new Terraform file with `project` resource (and `artifactory` resource as well). Also see [sample.tf](./sample.tf):

<details><summary>HCL Example</summary>

```terraform
# Required for Terraform 0.13 and up (https://www.terraform.io/upgrade-guides/0-13.html)
terraform {
  required_providers {
    artifactory = {
      source  = "registry.terraform.io/jfrog/artifactory"
      version = "2.6.24"
    }
    project = {
      source  = "registry.terraform.io/jfrog/project"
      version = "0.9.1"
    }
  }
}

provider "artifactory" {
  // supply ARTIFACTORY_USERNAME, ARTIFACTORY_PASSWORD and ARTIFACTORY_URL as env vars
}

provider "project" {
  // supply PROJECT_URL and PROJECT_ACCESS_TOKEN as env vars
}

variable "qa_roles" {
  type    = list(string)
  default = ["READ_REPOSITORY", "READ_RELEASE_BUNDLE", "READ_BUILD", "READ_SOURCES_PIPELINE", "READ_INTEGRATIONS_PIPELINE", "READ_POOLS_PIPELINE", "TRIGGER_PIPELINE"]
}

variable "devop_roles" {
  type    = list(string)
  default = ["READ_REPOSITORY", "ANNOTATE_REPOSITORY", "DEPLOY_CACHE_REPOSITORY", "DELETE_OVERWRITE_REPOSITORY", "TRIGGER_PIPELINE", "READ_INTEGRATIONS_PIPELINE", "READ_POOLS_PIPELINE", "MANAGE_INTEGRATIONS_PIPELINE", "MANAGE_SOURCES_PIPELINE", "MANAGE_POOLS_PIPELINE", "READ_BUILD", "ANNOTATE_BUILD", "DEPLOY_BUILD", "DELETE_BUILD", ]
}

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

  member {
    name  = "user1"
    roles = ["Developer", "Project Admin"]
  }

  member {
    name  = "user2"
    roles = ["Developer"]
  }

  group {
    name  = "qa"
    roles = ["qa"]
  }

  group {
    name  = "release"
    roles = ["Release Manager"]
  }

  role {
    name         = "qa"
    description  = "QA role"
    type         = "CUSTOM"
    environments = ["DEV"]
    actions      = var.qa_roles
  }

  role {
    name         = "devop"
    description  = "DevOp role"
    type         = "CUSTOM"
    environments = ["DEV", "PROD"]
    actions      = var.devop_roles
  }

  repos = ["docker-local", "npm-remote"]

  depends_on = [
    artifactory_user.user1,
    artifactory_user.user2,
    artifactory_group.qa-group,
    artifactory_group.release-group,
    artifactory_local_docker_v2_repository.docker-local,
    artifactory_remote_npm_repository.npm-remote,
  ]
}
```
</details>

Initialize Terrform:
```sh
$ terraform init
```

Plan (or Apply):
```sh
$ terraform plan
```

Detailed documentation of the resource and attributes are on [Terraform Registry](https://registry.terraform.io/providers/jfrog/project/latest/docs).

## License requirements:

This provider requires access to the APIs, which are only available in the _licensed_ pro and enterprise editions.
You can determine which license you have by accessing the following URL
`${host}/artifactory/api/system/licenses/`

You can either access it via api, or web browser - it does require admin level credentials, but it's one of the few APIs that will work without a license (side node: you can also install your license here with a `POST`)

```bash
curl -sL ${host}/artifactory/api/system/licenses/ | jq .
{
  "type" : "Enterprise Plus Trial",
  "validThrough" : "Jan 29, 2022",
  "licensedTo" : "JFrog Ltd"
}
```

The following 3 license types (`jq .type`) do **NOT** support APIs:
- Community Edition for C/C++
- JCR Edition
- OSS

## Limitations of functionality

Currently this provider does not support the followings:
- Xray support for the project

## Versioning

In general, this project follows [semver](https://semver.org/) as closely as we can for tagging releases of the package. We've adopted the following versioning policy:

* We increment the **major version** with any incompatible change to functionality, including changes to the exported Go API surface or behavior of the API.
* We increment the **minor version** with any backwards-compatible changes to functionality.
* We increment the **patch version** with any backwards-compatible bug fixes.

## Contributors

See the [contribution guide](CONTRIBUTIONS.md).

## License

Copyright (c) 2022 JFrog.

Apache 2.0 licensed, see [LICENSE][LICENSE] file.

[LICENSE]: ./LICENSE
