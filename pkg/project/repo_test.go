package project

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccProject_repo(t *testing.T) {
	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(10))

	repo1 := fmt.Sprintf("repo%s", strings.ToLower(randSeq(6)))
	repo2 := fmt.Sprintf("repo%s", strings.ToLower(randSeq(6)))

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repo1":       repo1,
		"repo2":       repo2,
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

			use_project_repository_resource = false

			repos = ["{{ .repo1 }}"]
		}
	`, params)

	addRepoConfig := util.ExecuteTemplate("TestAccProjectRepo", `
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

			repos = ["{{ .repo1 }}", "{{ .repo2 }}"]
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
		PreCheck: func() {
			testAccPreCheck(t)
			createTestRepo(t, repo1)
			createTestRepo(t, repo2)
		},
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			deleteTestRepo(t, repo1)
			deleteTestRepo(t, repo2)
			resp, err := verifyProject(id, request)

			return resp, err
		}),
		ProviderFactories: ProviderFactories,
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

	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(10))

	getRandomRepoNames := func(repoCount int) []string {
		var repoNames []string
		for i := 0; i < repoCount; i++ {
			repoNames = append(repoNames, fmt.Sprintf("%s%s", repoNameInitial, randSeq(10)))
		}
		return repoNames
	}

	randomRepoNames := getRandomRepoNames(numRepos)

	preCheck := func(t *testing.T, repoNames []string) func() {
		return func() {
			testAccPreCheck(t)
			for _, repoName := range repoNames {
				createTestRepo(t, repoName)
			}
		}
	}

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repos":       randomRepoNames,
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

			repos = [{{range $idx, $elem := .repos}}{{if $idx}},{{end}}"{{ $elem }}"{{end}}]
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
		PreCheck: preCheck(t, randomRepoNames),
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			for _, repoName := range randomRepoNames {
				deleteTestRepo(t, repoName)
			}
			resp, err := verifyProject(id, request)
			return resp, err
		}),
		ProviderFactories: ProviderFactories,
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
	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(10))

	repo := fmt.Sprintf("repo%s", strings.ToLower(randSeq(6)))

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repo":        repo,
	}

	initialConfig := util.ExecuteTemplate("TestAccProjectRepoUnassignNonexistantRepo", `
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
		PreCheck: func() {
			testAccPreCheck(t)
			createTestRepo(t, repo)
		},
		CheckDestroy:      verifyDeleted(resourceName, verifyProject),
		ProviderFactories: ProviderFactories,
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
				// PreConfig is used to delete the repo out-of-band from TF.
				PreConfig: func() {
					deleteTestRepo(t, repo)
				},
				Config: initialConfig,
				// SkipFunc is called after PreConfig but before applying the Config.
				// https://github.com/hashicorp/terraform-plugin-sdk/blob/main/helper/resource/testing_new.go#L133
				//
				// We are skipping this checks because there's nothing to check on the resource.
				// We want to verify the resource is deleted without error which resource.TestCase
				// will do that for us.
				SkipFunc: func() (bool, error) {
					return true, nil
				},
			},
		},
	})
}
