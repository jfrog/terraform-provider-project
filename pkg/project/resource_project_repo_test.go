package project

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"strconv"
	"strings"
	"testing"
)

func TestAccProjectRepo(t *testing.T) {
	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(6))

	repo1 := "repo1"
	repo2 := "repo2"

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repo1":       repo1,
		"repo2":       repo2,
	}

	initialConfig := executeTemplate("TestAccProjectRepo", `
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

	addRepoConfig := executeTemplate("TestAccProjectRepo", `
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

	noReposConfig := executeTemplate("TestAccProjectRepo", `
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
					resource.TestCheckResourceAttr(resourceName, "repos.0", repo1),
					resource.TestCheckResourceAttr(resourceName, "repos.1", repo2),
				),
			},
			{
				Config: noReposConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckNoResourceAttr(resourceName, "repos"),
				),
			},
		},
	})
}

/*
Test to assign large number of repositories to a project
*/
func TestAccAssignMultipleReposInProject(t *testing.T) {

	const numRepos = 200
	const repoNameInitial = "repo-"

	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(6))

	repoNamesStr := func(repoCount int) string {
		var repoNames []string
		for i := 0; i < repoCount; i++ {
			repoNames = append(repoNames, repoNameInitial+strconv.Itoa(i))
		}
		quote := "\""
		if len(repoNames) > 0 {
			return fmt.Sprintf("%[1]s%[2]s%[1]s", quote, strings.Join(repoNames[:], "\",\""))
		} else {
			return fmt.Sprintf("%v", strings.Join(repoNames[:], "\",\""))
		}
	}

	preCheck := func(t *testing.T, numRepo int) func() {
		return func() {
			testAccPreCheck(t)
			for i := 0; i < numRepo; i++ {
				createTestRepo(t, repoNameInitial+strconv.Itoa(i))
			}
		}
	}

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"repos":       repoNamesStr(numRepos),
	}

	initialConfig := executeTemplate("TestAccProjectRepo", `
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

	addRepoConfig := executeTemplate("TestAccProjectRepo", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
			repos = [{{ .repos }}]
		}
	`, params)

	noReposConfig := executeTemplate("TestAccProjectRepo", `
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
		PreCheck: preCheck(t, numRepos),
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			for i := 0; i < numRepos; i++ {
				deleteTestRepo(t, repoNameInitial+strconv.Itoa(i))
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
					//resource.TestCheckResourceAttr(resourceName, "repos.0", repo1),
				),
			},
			{
				Config: addRepoConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", strconv.Itoa(numRepos)),
					//resource.TestCheckResourceAttr(resourceName, "repos.0", repo1),
					//resource.TestCheckResourceAttr(resourceName, "repos.1", repo2),
				),
			},
			{
				Config: noReposConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckNoResourceAttr(resourceName, "repos"),
				),
			},
		},
	})
}
