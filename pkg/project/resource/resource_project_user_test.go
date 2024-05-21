package project_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	project "github.com/jfrog/terraform-provider-project/pkg/project/resource"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccProjectUser(t *testing.T) {
	projectName := fmt.Sprintf("tftestprojects%s", acctest.RandSeq(10))
	projectKey := strings.ToLower(acctest.RandSeq(10))

	username := fmt.Sprintf("user%s", strings.ToLower(acctest.RandSeq(5)))
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

			use_project_user_resource = true
		}
		
		resource "project_user" "{{ .username }}" {
			project_key = project.{{ .project_name }}.key
			name = artifactory_managed_user.{{ .username }}.name
			roles = {{ .roles }}
		}
	`

	config := util.ExecuteTemplate("TestAccProjectUser", template, params)

	updateParams := map[string]interface{}{
		"project_name": params["project_name"],
		"project_key":  params["project_key"],
		"username":     params["username"],
		"email":        params["email"],
		"roles":        `["Developer"]`,
	}

	configUpdated := util.ExecuteTemplate("TestAccProjectUser", template, updateParams)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyProjectUser(username, projectKey, request)
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source:            "jfrog/artifactory",
				VersionConstraint: "10.1.4",
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
	projectName := fmt.Sprintf("tftestprojects%s", acctest.RandSeq(10))
	projectKey := strings.ToLower(acctest.RandSeq(10))

	username := fmt.Sprintf("not_existing%s", strings.ToLower(acctest.RandSeq(5)))
	email := username + "@tempurl.org"

	params := map[string]interface{}{
		"project_name": projectName,
		"project_key":  projectKey,
		"username":     username,
		"email":        email,
		"roles":        `["Developer","Project Admin"]`,
	}

	template := `
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

			use_project_user_resource = true
		}

		resource "project_user" "{{ .username }}" {
			project_key = project.{{ .project_name }}.key
			name = "{{ .username }}"
			roles = {{ .roles }}
			ignore_missing_user = false
		}		
	`

	config := util.ExecuteTemplate("TestAccProjectUser", template, params)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`user '.*' not found, project membership not created.*`),
			},
		},
	})
}

func TestAccProjectMember_missing_user_ignored(t *testing.T) {
	projectName := fmt.Sprintf("tftestprojects%s", acctest.RandSeq(10))
	projectKey := strings.ToLower(acctest.RandSeq(10))

	username := fmt.Sprintf("not_existing%s", strings.ToLower(acctest.RandSeq(5)))
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

			use_project_user_resource = true
		}

		resource "project_user" "{{ .username }}" {
			project_key = project.{{ .project_name }}.key
			name = "{{ .username }}"
			roles = {{ .roles }}
			ignore_missing_user = true
		}
	`

	config := util.ExecuteTemplate("TestAccProjectUser", template, params)
	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyProjectUser(username, projectKey, request)
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
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
		Get(project.ProjectUsersUrl)
}
