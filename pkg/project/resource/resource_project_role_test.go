package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	project "github.com/jfrog/terraform-provider-project/pkg/project/resource"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccProjectRole_full(t *testing.T) {
	name := acctest.RandSeq(20)
	resourceName := fmt.Sprintf("project_role.%s", name)
	projectKey := strings.ToLower(acctest.RandSeq(10))

	template := `
		resource "project" "{{ .project_name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
		}

		resource "project_role" "{{ .name }}" {
			name = "{{ .name }}"
			type = "{{ .type }}"
			project_key = project.{{ .project_name }}.key
			
			environments = ["{{ .environment }}"]
			actions = ["{{ .action }}"]
		}
	`

	testData := map[string]string{
		"name":         name,
		"project_name": projectKey,
		"project_key":  projectKey,
		"type":         "CUSTOM",
		"environment":  "DEV",
		"action":       "READ_REPOSITORY",
	}

	testUpdatedData := map[string]string{
		"name":         name,
		"project_name": projectKey,
		"project_key":  projectKey,
		"type":         "CUSTOM",
		"environment":  "PROD",
		"action":       "ANNOTATE_REPOSITORY",
	}

	config := util.ExecuteTemplate("TestAccProjectRole", template, testData)
	updatedConfig := util.ExecuteTemplate("TestAccProjectRole", template, testUpdatedData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyRole(id, projectKey, request)
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", testData["name"]),
					resource.TestCheckResourceAttr(resourceName, "project_key", testData["project_key"]),
					resource.TestCheckResourceAttr(resourceName, "type", testData["type"]),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0", testData["environment"]),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0", testData["action"]),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", testUpdatedData["name"]),
					resource.TestCheckResourceAttr(resourceName, "project_key", testUpdatedData["project_key"]),
					resource.TestCheckResourceAttr(resourceName, "type", testUpdatedData["type"]),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0", testUpdatedData["environment"]),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0", testUpdatedData["action"]),
				),
			},
		},
	})
}

func TestAccProjectRole_conflict_with_project(t *testing.T) {
	name := acctest.RandSeq(20)
	resourceName := fmt.Sprintf("project_role.%s", name)
	projectKey := strings.ToLower(acctest.RandSeq(10))

	template := `
		resource "project" "{{ .project_name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_role_resource = false
		}

		resource "project_role" "{{ .name }}" {
			name = "{{ .name }}"
			type = "{{ .type }}"
			project_key = project.{{ .project_name }}.key
			
			environments = ["{{ .environment }}"]
			actions = ["{{ .action }}"]
		}
	`

	testData := map[string]string{
		"name":         name,
		"project_name": projectKey,
		"project_key":  projectKey,
		"type":         "CUSTOM",
		"environment":  "DEV",
		"action":       "READ_REPOSITORY",
	}

	config := util.ExecuteTemplate("TestAccProjectRole", template, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyRole(id, projectKey, request)
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", testData["name"]),
					resource.TestCheckResourceAttr(resourceName, "project_key", testData["project_key"]),
					resource.TestCheckResourceAttr(resourceName, "type", testData["type"]),
					resource.TestCheckResourceAttr(resourceName, "environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "environments.0", testData["environment"]),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "actions.0", testData["action"]),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func verifyRole(name, projectKey string, request *resty.Request) (*resty.Response, error) {
	return request.
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   name,
		}).
		Get(project.ProjectRoleUrl)
}
