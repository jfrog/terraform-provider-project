package project_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccProject_repo(t *testing.T) {
	name := "tftestprojects" + acctest.RandSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(acctest.RandSeq(10))

	repo1 := fmt.Sprintf("repo%s", strings.ToLower(acctest.RandSeq(6)))
	repo2 := fmt.Sprintf("repo%s", strings.ToLower(acctest.RandSeq(6)))

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repo1":       repo1,
		"repo2":       repo2,
	}

	initialConfig := util.ExecuteTemplate("TestAccProjectRepo", `
		resource "artifactory_local_generic_repository" "{{ .repo1 }}" {
			key = "{{ .repo1 }}"

			lifecycle {
				ignore_changes = [project_key]
			}
		}
		
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_repository_resource = false

			repos = [artifactory_local_generic_repository.{{ .repo1 }}.key]
		}
	`, params)

	addRepoConfig := util.ExecuteTemplate("TestAccProjectRepo", `
		resource "artifactory_local_generic_repository" "{{ .repo1 }}" {
			key = "{{ .repo1 }}"

			lifecycle {
				ignore_changes = [project_key]
			}
		}
		
		resource "artifactory_local_generic_repository" "{{ .repo2 }}" {
			key = "{{ .repo2 }}"

			lifecycle {
				ignore_changes = [project_key]
			}
		}
		
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_repository_resource = false

			repos = [
				artifactory_local_generic_repository.{{ .repo1 }}.key,
				artifactory_local_generic_repository.{{ .repo2 }}.key
			]
		}
	`, params)

	noReposConfig := util.ExecuteTemplate("TestAccProjectRepo", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_repository_resource = false
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             acctest.VerifyDeleted(resourceName, verifyProject),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "repos.0", repo1),
				),
			},
			{
				Config: addRepoConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "repos.*", repo1),
					resource.TestCheckTypeSetElemAttr(resourceName, "repos.*", repo2),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"use_project_role_resource",
					"use_project_user_resource",
					"use_project_group_resource",
					"use_project_repository_resource",
				},
			},
			{
				Config: noReposConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", "0"),
				),
			},
		},
	})
}

/*
Test to assign large number of repositories to a project
*/
func TestAccProject_repoAssignMultipleRepos(t *testing.T) {

	const numRepos = 5
	const repoNameInitial = "repo-"

	name := "tftestprojects" + acctest.RandSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(acctest.RandSeq(10))

	getRandomRepoNames := func(repoCount int) []string {
		var repoNames []string
		for i := 0; i < repoCount; i++ {
			repoNames = append(repoNames, fmt.Sprintf("%s%s", repoNameInitial, acctest.RandSeq(10)))
		}
		return repoNames
	}

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repos":       getRandomRepoNames(numRepos),
	}

	initialConfig := util.ExecuteTemplate("TestAccProjectRepo", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
		}
	`, params)

	addRepoConfig := util.ExecuteTemplate("TestAccProjectRepo", `
	{{ range $repoName := .repos }}
		resource "artifactory_local_generic_repository" "{{ $repoName }}" {
			key = "{{ $repoName }}"

			lifecycle {
				ignore_changes = [project_key]
			}
		}
	{{ end }}

		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_repository_resource = false

			repos = [{{range $idx, $elem := .repos}}{{if $idx}},{{end}}artifactory_local_generic_repository.{{ $elem }}.key{{end}}]
		}
	`, params)

	noReposConfig := util.ExecuteTemplate("TestAccProjectRepo", `
	{{ range $repoName := .repos }}
		resource "artifactory_local_generic_repository" "{{ $repoName }}" {
			key = "{{ $repoName }}"

			lifecycle {
				ignore_changes = [project_key]
			}
		}
	{{ end }}
	
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_repository_resource = false
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             acctest.VerifyDeleted(resourceName, verifyProject),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckNoResourceAttr(resourceName, "repos"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", "0"),
				),
			},
			{
				Config: addRepoConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", strconv.Itoa(numRepos)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"use_project_role_resource",
					"use_project_user_resource",
					"use_project_group_resource",
					"use_project_repository_resource",
				},
			},
			{
				Config: noReposConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", "0"),
				),
			},
		},
	})
}

func TestAccProject_repoUnassignNonexistantRepo(t *testing.T) {
	name := "tftestprojects" + acctest.RandSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(acctest.RandSeq(10))

	repo := fmt.Sprintf("repo%s", strings.ToLower(acctest.RandSeq(6)))

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repo":        repo,
	}

	initialConfig := util.ExecuteTemplate("TestAccProjectRepoUnassignNonexistantRepo", `
		resource "artifactory_local_generic_repository" "{{ .repo }}" {
			key = "{{ .repo }}"

			lifecycle {
				ignore_changes = [project_key]
			}
		}

		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_repository_resource = false

			repos = ["{{ .repo }}"]

			depends_on = [
				artifactory_local_generic_repository.{{ .repo }}
			]
		}
	`, params)

	updatedConfig := util.ExecuteTemplate("TestAccProjectRepoUnassignNonexistantRepo", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_repository_resource = false

			repos = ["{{ .repo }}"]
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             acctest.VerifyDeleted(resourceName, verifyProject),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "repos.0", repo),
				),
			},
			{
				Config: updatedConfig,
			},
		},
	})
}
