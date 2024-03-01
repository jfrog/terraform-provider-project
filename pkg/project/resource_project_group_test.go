package project

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/jfrog/terraform-provider-shared/test"
)

func TestAccProjectGroup(t *testing.T) {
	projectName := fmt.Sprintf("tftestprojects%s", randSeq(10))
	projectKey := strings.ToLower(randSeq(10))

	group := fmt.Sprintf("group%s", strings.ToLower(randSeq(5)))

	resourceName := "project_group." + group

	params := map[string]interface{}{
		"project_name": projectName,
		"project_key":  projectKey,
		"group":        group,
		"roles":        `["Developer","Project Admin"]`,
	}

	template := `
		resource "artifactory_group" "{{ .group }}" {
			name = "{{ .group }}"
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

		resource "project_group" "{{ .group }}" {
			project_key = project.{{ .project_name }}.key
			name = artifactory_group.{{ .group }}.name
			roles = {{ .roles }}
		}
	`

	config := test.ExecuteTemplate("TestAccProjectGroup", template, params)

	updateParams := map[string]interface{}{
		"project_name": params["project_name"],
		"project_key":  params["project_key"],
		"group":        params["group"],
		"roles":        `["Developer"]`,
	}

	configUpdated := test.ExecuteTemplate("TestAccProjectGroup", template, updateParams)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyProjectUser(group, projectKey, request)
		}),
		ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "name", group),
					resource.TestCheckResourceAttr(resourceName, "roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "roles.0", "Developer"),
					resource.TestCheckResourceAttr(resourceName, "roles.1", "Project Admin"),
				),
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "name", group),
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

func verifyProjectGroup(name string, projectKey string, request *resty.Request) (*resty.Response, error) {
	return request.
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"name":       name,
		}).
		Get(projectGroupsUrl)
}
