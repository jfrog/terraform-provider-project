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

## Build the Provider

Simply run `make install` - this will compile the provider and install it to `~/terraform.d`. When running this, it will take the current tag and bump it 1 minor version. It does not actually create a new tag (that is `make release`). If you wish to use the locally installed provider, make sure your TF script refers to the new version number.

Requirements:
- [Terraform](https://www.terraform.io/downloads.html) 0.13
- [Go](https://golang.org/doc/install) 1.15+ (to build the provider plugin)

### Building on macOS

This provider uses [GNU sed](https://www.gnu.org/software/sed/) as part of the build toolchain, in both Linux and macOS. This provides consistency across OSes.

If you are building this on macOS, you have two options:
- Install [gnu-sed using brew](https://formulae.brew.sh/formula/gnu-sed), OR
- Use a Linux Docker image/container

#### Using gnu-sed

After installing with brew, get the GNU sed information:

```sh
$ brew info gnu-sed
```

You should see something like:
```
GNU "sed" has been installed as "gsed".
If you need to use it as "sed", you can add a "gnubin" directory
to your PATH from your bashrc like:

     PATH="$(brew --prefix)/opt/gnu-sed/libexec/gnubin:$PATH"
```

Add the `gnubin` directory to your `.bashrc` or `.zshrc` per instruction so that `sed` command uses gnu-sed.

## Testing

How to run the tests isn't obvious.

First, you need a running instance of the JFrog platform (RT and XR). However, there is no currently supported Dockerized, local version. You can ask for an instance to test against in as part of your PR or by messaging the maintainer in gitter.

Alternatively, you can run the file [scripts/run-artifactory.sh](scripts/run-artifactory.sh), which, if you have a license file in the same directory called `artifactory.lic`, you can start just an artifactory instance. The license is not supplied, but a [30 day trial license can be freely obtained](https://jfrog.com/start-free/#hosted) and will allow local development.

Then, you have to set some environment variables as this is how the acceptance tests pick up their config

```bash
PROJECT_URL=http://localhost:8081
PROJECT_ACCESS_TOKEN=...
TF_ACC=true
```

A crucial env var to set is `TF_ACC=true` - you can literally set `TF_ACC` to anything you want, so long as it's set. The acceptance tests use terraform testing libraries that, if this flag isn't set, will skip all tests. See [Terraform doc](https://www.terraform.io/docs/extend/testing/acceptance-tests/index.html#running-acceptance-tests).

You can then run the tests with:
```sh
$ go test -v ./pkg/...
```

**DO NOT** omit the `-v` - terraform testing needs this (don't ask me why). This will recursively run all tests, including acceptance tests.

## Debugging

### Debugger-based debugging

Debugging a terraform provider is not straightforward. Terraform forks your provider as a separate process and then connects to it via RPC. Normally, when debugging, you would start the process to debug directly. However, with the terraform + go architecture, this isn't possible. So, you need to run terraform as you normally would and attach to the provider process by getting its pid. This would be really tricky considering how fast the process can come up and be down. So, you need to actually halt the provider and have it wait for your debugger to attach.

Having said all that, here are the steps:
1. Install [delve](https://github.com/go-delve/delve)
2. Keep in mind that terraform will parallel process if it can, and it will start new instances of the TF provider process when running apply between the plan and confirmation.
   Add a snippet of go code to the close to where you need to break where in you install a busy sleep loop:
```go
	debug := true
	for debug {
		time.Sleep(time.Second) // set breakpoint here
	}
```
Then set a breakpoint inside the loop. Once you have attached to the process you can set the `debug` value to `false`, thus breaking the sleep loop and allow you to continue.
2. Compile the provider with debug symbology (`go build -gcflags "all=-N -l"`)
3. Install the provider (change as needed for your version)
```bash
# this will bump your version by 1 so it doesn't download from TF. Make sure you update any test scripts accordingly
make install
```
4. Run your provider: `terraform init && terraform plan` - it will start in this busy sleep loop.
5. In a separate shell, find the `PID` of the provider that got forked
`pgrep terraform-provider-projects`
6. Then, attach the debugger to that pid: `dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient attach $pid`
A 1-liner for this whole process is:
`dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient attach $(pgrep terraform-provider-projects)`
7. In intellij, setup a remote go debugging session (the default port is `2345`, but make sure it's set.) And click the `debug` button
8. Your editor should immediately break at the breakpoint from step 2. At this point, in the watch window, edit the `debug` value and set it to false, and allow the debugger to continue. Be ready for your debugging as this will release the provider and continue executing normally.

You will need to repeat steps 4-8 every time you want to debug

### Log-based debugging

You can [turn on logging](https://www.terraform.io/docs/extend/debugging.html#turning-on-logging) for debug purpose by setting env var `TF_LOG` to `DEBUG` or `TRACE`, i.e.

```sh
export TF_LOG=DEBUG
```

Then use `log.Printf()` to print the data you want to the console.

**Note** that you must include the log level as the prefix to the log message, e.g.

```go
tflog.Debug("some thing happened")
```

## Registry documentation generation

All the registry documentation is generated using [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). If you make any changes to the resource schemas, you will need to re-generate documentation.

Install [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs#installation), then run:
```sh
$ make doc
```

## Versioning

In general, this project follows [semver](https://semver.org/) as closely as we can for tagging releases of the package. We've adopted the following versioning policy:

* We increment the **major version** with any incompatible change to functionality, including changes to the exported Go API surface or behavior of the API.
* We increment the **minor version** with any backwards-compatible changes to functionality.
* We increment the **patch version** with any backwards-compatible bug fixes.

## Contributors

Pull requests, issues and comments are welcomed. For pull requests:

* Add tests for new features and bug fixes
* Follow the existing style
* Separate unrelated changes into multiple pull requests

See the existing issues for things to start contributing.

For bigger changes, make sure you start a discussion first by creating an issue and explaining the intended change.

JFrog requires contributors to sign a Contributor License Agreement, known as a CLA. This serves as a record stating that the contributor is entitled to contribute the code/documentation/translation to the project and is willing to have it used in distributions and derivative works (or is willing to transfer ownership).

## License

Copyright (c) 2022 JFrog.

Apache 2.0 licensed, see [LICENSE][LICENSE] file.

[LICENSE]: ./LICENSE
