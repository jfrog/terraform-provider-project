package project

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/jfrog/terraform-provider-shared/test"
)

func TestAccProjectRepo(t *testing.T) {
	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(6))

	repo1 := fmt.Sprintf("repo%d", test.RandomInt())
	repo2 := fmt.Sprintf("repo%d", test.RandomInt())

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repo1":       repo1,
		"repo2":       repo2,
	}

	initialConfig := test.ExecuteTemplate("TestAccProjectRepo", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			repos = ["{{ .repo1 }}"]
		}
	`, params)

	addRepoConfig := test.ExecuteTemplate("TestAccProjectRepo", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			repos = ["{{ .repo1 }}", "{{ .repo2 }}"]
		}
	`, params)

	noReposConfig := test.ExecuteTemplate("TestAccProjectRepo", `
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
		ProviderFactories: testAccProviders(),
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
func TestAccAssignMultipleReposInProject(t *testing.T) {

	const numRepos = 5
	const repoNameInitial = "repo-"

	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(6))

	randomRepoNames := func(repoCount int) []string {
		var repoNames []string
		for i := 0; i < repoCount; i++ {
			repoNames = append(repoNames, fmt.Sprintf("%s%s", repoNameInitial, randSeq(10)))
		}
		return repoNames
	}(numRepos)

	repoNamesStr := func(repoNames []string) string {
		jsonByteArr, err := json.Marshal(repoNames)
		if err != nil {
			return "[]"
		}
		return string(jsonByteArr)
	}

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
		"repos":       repoNamesStr(randomRepoNames),
	}

	initialConfig := test.ExecuteTemplate("TestAccProjectRepo", `
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

	addRepoConfig := test.ExecuteTemplate("TestAccProjectRepo", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
			repos = {{ .repos }}
		}
	`, params)

	noReposConfig := test.ExecuteTemplate("TestAccProjectRepo", `
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

	resource.Test(t, resource.TestCase{
		PreCheck: preCheck(t, randomRepoNames),
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			for _, repoName := range randomRepoNames {
				deleteTestRepo(t, repoName)
			}
			resp, err := verifyProject(id, request)
			return resp, err
		}),
		ProviderFactories: testAccProviders(),
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
