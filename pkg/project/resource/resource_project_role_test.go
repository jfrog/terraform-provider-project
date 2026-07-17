package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	project "github.com/jfrog/terraform-provider-project/pkg/project/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccProjectRole_UpgradeFromSDKv2(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, roleName := testutil.MkNames("test-project-role-", "project_role")

	projectKey := strings.ToLower(acctest.RandSeq(10))

	// Legacy template for Step 1: project provider 1.6.1 did not have manage_remote_repository
	legacyTemplate := `
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

	// Template for Step 2: current provider adds manage_remote_repository. Set it
	// explicitly to match the value Artifactory assigns to the project created by
	// the legacy provider, so the post-upgrade plan is empty.
	template := `
		resource "project" "{{ .project_name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			admin_privileges {
				manage_members = true
				manage_resources = true
				manage_remote_repository = true
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
		"name":         roleName,
		"project_name": projectName,
		"project_key":  projectKey,
		"type":         "CUSTOM",
		"environment":  "DEV",
		"action":       "READ_REPOSITORY",
	}

	legacyConfig := util.ExecuteTemplate("TestAccProjectRole", legacyTemplate, testData)
	config := util.ExecuteTemplate("TestAccProjectRole", template, testData)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"project": {
						Source:            "jfrog/project",
						VersionConstraint: "1.6.1",
					},
				},
				Config: legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", testData["name"]),
					resource.TestCheckResourceAttr(fqrn, "project_key", testData["project_key"]),
					resource.TestCheckResourceAttr(fqrn, "type", testData["type"]),
					resource.TestCheckResourceAttr(fqrn, "environments.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "environments.0", testData["environment"]),
					resource.TestCheckResourceAttr(fqrn, "actions.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "actions.0", testData["action"]),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 false,
				ConfigPlanChecks:         testutil.ConfigPlanChecks(fqrn),
			},
		},
	})
}

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
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
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
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s:%s", testUpdatedData["project_key"], testUpdatedData["name"]),
				ImportStateVerify: true,
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
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
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
