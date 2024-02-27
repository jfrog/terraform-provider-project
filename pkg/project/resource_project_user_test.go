package project

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/jfrog/terraform-provider-shared/test"
)

func TestAccProjectUser(t *testing.T) {
	projectName := fmt.Sprintf("tftestprojects%s", randSeq(10))
	projectKey := strings.ToLower(randSeq(6))

	username := "user1"
	email := username + "@tempurl.org"

	resourceName := "project_user." + username

	params := map[string]interface{}{
		"project_name": projectName,
		"project_key":  projectKey,
		"username":     username,
		"email":        email,
		"roles":        `["Developer","Project Admin"]`,
	}

	template := `
		resource "artifactory_managed_user" "{{ .username }}" {
			name     = "{{ .username }}"
			email    = "{{ .email }}"
			password = "Password1!"
			admin    = false
		}

		resource "project" "{{ .project_name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
			max_storage_in_gibibytes = 1
			block_deployments_on_limit = true
			email_notification = false

			lifecycle {
				ignore_changes = ["member"]
			}

			depends_on = [
				artifactory_managed_user.{{ .username }}
			]
		}
		
		resource "project_user" "{{ .username }}" {
			project_key = "{{ .project_key }}"
			name = "{{ .username }}"
			roles = {{ .roles }}

			depends_on = [
				project.{{ .project_name }}
			]
		}
	`

	config := test.ExecuteTemplate("TestAccProjectUser", template, params)

	updateParams := map[string]interface{}{
		"project_name": params["project_name"],
		"project_key":  params["project_key"],
		"username":     params["username"],
		"email":        params["email"],
		"roles":        `["Developer"]`,
	}

	configUpdated := test.ExecuteTemplate("TestAccProjectUser", template, updateParams)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyProjectUser(username, projectKey, request)
		}),
		ProviderFactories: testAccProviders(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source:            "jfrog/artifactory",
				VersionConstraint: "10.1.3",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "name", username),
					resource.TestCheckResourceAttr(resourceName, "ignore_missing_user", "false"),
					resource.TestCheckResourceAttr(resourceName, "roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "roles.0", "Developer"),
					resource.TestCheckResourceAttr(resourceName, "roles.1", "Project Admin"),
				),
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "name", username),
					resource.TestCheckResourceAttr(resourceName, "ignore_missing_user", "false"),
					resource.TestCheckResourceAttr(resourceName, "roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "roles.0", "Developer"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccProjectUser_missing_user_fails(t *testing.T) {
	projectName := fmt.Sprintf("tftestprojects%s", randSeq(10))
	projectKey := strings.ToLower(randSeq(6))

	username := "not_existing"
	email := username + "@tempurl.org"

	resourceName := "project_user." + username

	params := map[string]interface{}{
		"project_name": projectName,
		"project_key":  projectKey,
		"username":     username,
		"email":        email,
		"roles":        `["Developer","Project Admin"]`,
	}

	template := `
		resource "project_user" "{{ .username }}" {
			project_key = "{{ .project_key }}"
			name = "{{ .username }}"
			roles = {{ .roles }}
			ignore_missing_user = false
		}

		resource "project" "{{ .project_name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
			max_storage_in_gibibytes = 1
			block_deployments_on_limit = true
			email_notification = false

			lifecycle {
				ignore_changes = ["member"]
			}
		}
	`

	config := test.ExecuteTemplate("TestAccProjectUser", template, params)
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyProjectUser(username, projectKey, request)
		}),
		ProviderFactories: testAccProviders(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source:            "jfrog/artifactory",
				VersionConstraint: "10.1.3",
			},
		},
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`user '.*' not found, project membership not created.*`),
			},
		},
	})
}

func TestAccProjectMember_missing_user_ignored(t *testing.T) {
	projectName := fmt.Sprintf("tftestprojects%s", randSeq(10))
	projectKey := strings.ToLower(randSeq(6))

	username := "not_existing"
	email := username + "@tempurl.org"

	resourceName := "project_user." + username

	params := map[string]interface{}{
		"project_name": projectName,
		"project_key":  projectKey,
		"username":     username,
		"email":        email,
		"roles":        `["Developer","Project Admin"]`,
	}

	template := `
		resource "project_user" "{{ .username }}" {
			project_key = "{{ .project_key }}"
			name = "{{ .username }}"
			roles = {{ .roles }}
			ignore_missing_user = true
		}

		resource "project" "{{ .project_name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
			max_storage_in_gibibytes = 1
			block_deployments_on_limit = true
			email_notification = false

			lifecycle {
				ignore_changes = ["member"]
			}
		}
	`

	config := test.ExecuteTemplate("TestAccProjectUser", template, params)
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyProjectUser(username, projectKey, request)
		}),
		ProviderFactories: testAccProviders(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source:            "jfrog/artifactory",
				VersionConstraint: "10.1.3",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "name", username),
					resource.TestCheckResourceAttr(resourceName, "ignore_missing_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "roles.0", "Developer"),
					resource.TestCheckResourceAttr(resourceName, "roles.1", "Project Admin"),
				)},
		},
	})
}

func verifyProjectUser(name string, projectKey string, request *resty.Request) (*resty.Response, error) {
	return request.
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"name":       name,
		}).
		Get(projectUsersUrl)
}
