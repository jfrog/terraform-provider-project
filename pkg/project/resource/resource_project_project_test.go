package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	"github.com/jfrog/terraform-provider-shared/util"
)

// TestAccProjectProject_basic verifies the namespaced `project_project` resource
// can create and manage a project, mirroring the legacy `project` resource.
func TestAccProjectProject_basic(t *testing.T) {
	name := fmt.Sprintf("tftestprojectproject%s", acctest.RandSeq(10))
	resourceName := fmt.Sprintf("project_project.%s", name)

	params := map[string]interface{}{
		"name":        name,
		"project_key": fmt.Sprintf("key%s", strings.ToLower(acctest.RandSeq(5))),
	}

	config := util.ExecuteTemplate("TestAccProjectProject", `
		resource "project_project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				manage_remote_repository = true
				index_resources = true
			}
			max_storage_in_gibibytes = 10
			block_deployments_on_limit = false
			email_notification = true
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             acctest.VerifyDeleted(resourceName, verifyProject),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", params["project_key"].(string)),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "max_storage_in_gibibytes", "10"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     params["project_key"].(string),
				ImportStateVerify: true,
				// The `use_project_*_resource` toggles are computed convenience
				// flags that the import (GET) response does not carry; this
				// matches the legacy `project` resource's import behavior.
				ImportStateVerifyIgnore: []string{
					"use_project_group_resource",
					"use_project_repository_resource",
					"use_project_role_resource",
					"use_project_user_resource",
				},
			},
		},
	})
}

// TestAccProjectProject_migrateFromProject verifies that an existing `project`
// resource can be migrated to the namespaced `project_project` resource using a
// `moved` block without destroying and recreating the underlying project. This
// is the migration path recommended in the deprecation message. See issue #210.
func TestAccProjectProject_migrateFromProject(t *testing.T) {
	name := fmt.Sprintf("tftestprojectmigrate%s", acctest.RandSeq(10))
	legacyResourceName := fmt.Sprintf("project.%s", name)
	newResourceName := fmt.Sprintf("project_project.%s", name)

	params := map[string]interface{}{
		"name":        name,
		"project_key": fmt.Sprintf("key%s", strings.ToLower(acctest.RandSeq(5))),
	}

	legacyConfig := util.ExecuteTemplate("TestAccProjectMigrate", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				manage_remote_repository = true
				index_resources = true
			}
			max_storage_in_gibibytes = 10
			block_deployments_on_limit = false
			email_notification = true
		}
	`, params)

	migratedConfig := util.ExecuteTemplate("TestAccProjectMigrate", `
		resource "project_project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				manage_remote_repository = true
				index_resources = true
			}
			max_storage_in_gibibytes = 10
			block_deployments_on_limit = false
			email_notification = true
		}

		moved {
			from = project.{{ .name }}
			to   = project_project.{{ .name }}
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             acctest.VerifyDeleted(newResourceName, verifyProject),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: legacyConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(legacyResourceName, "key", params["project_key"].(string)),
				),
			},
			{
				Config: migratedConfig,
				// The moved block should relocate state to the new address with
				// no destroy/create and no attribute changes.
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(newResourceName, plancheck.ResourceActionNoop),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(newResourceName, "key", params["project_key"].(string)),
					resource.TestCheckResourceAttr(newResourceName, "display_name", name),
				),
			},
		},
	})
}
