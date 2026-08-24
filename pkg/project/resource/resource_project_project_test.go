package project_test

import (
	"fmt"
	"regexp"
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

// TestAccProjectProject_migrateFromOlderSchemaVersion covers the journey for a
// practitioner whose state predates schema version 4: the move is refused with
// an actionable error, one apply on the current version upgrades the state, and
// the move then succeeds as a pure no-op.
//
// Provider 1.4.0 writes schema version 3 -- it added `use_project_user_resource`
// and `use_project_group_resource`, while 1.5.0 added
// `use_project_repository_resource` and bumped the schema to 4. Its
// `admin_privileges` block has no `manage_remote_repository` attribute (that
// arrived in 1.9.6), so the first step omits it.
//
// The test relies on Terraform handing the state mover the raw stored state
// rather than running the target's UpgradeState upgraders first. That is what
// makes both halves observable: with the `moved` block present the mover sees
// version 3 and refuses, and without it a normal refresh upgrades to version 4.
func TestAccProjectProject_migrateFromOlderSchemaVersion(t *testing.T) {
	name := fmt.Sprintf("tftestprojectupgrade%s", acctest.RandSeq(10))
	legacyResourceName := fmt.Sprintf("project.%s", name)
	newResourceName := fmt.Sprintf("project_project.%s", name)

	params := map[string]interface{}{
		"name":        name,
		"project_key": fmt.Sprintf("key%s", strings.ToLower(acctest.RandSeq(5))),
	}

	// Only the attributes provider 1.4.0 knows about.
	schemaV3Config := util.ExecuteTemplate("TestAccProjectUpgradeV3", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
			max_storage_in_gibibytes = 10
			block_deployments_on_limit = false
			email_notification = true

			use_project_role_resource = true
			use_project_user_resource = true
			use_project_group_resource = true
		}
	`, params)

	// The same project under the current schema, still at the legacy address.
	// The toggles are set explicitly so the upgraded state does not depend on
	// schema defaults being applied over the upgraders' values.
	//
	// `manage_remote_repository` is set to true to match what the platform
	// reports for a project created before the attribute existed; asking for
	// false here produces a permanent diff.
	legacyCurrentConfig := util.ExecuteTemplate("TestAccProjectUpgradeLegacy", `
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

			use_project_role_resource = true
			use_project_user_resource = true
			use_project_group_resource = true
			use_project_repository_resource = true
		}
	`, params)

	movedConfig := util.ExecuteTemplate("TestAccProjectUpgradeMoved", `
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

			use_project_role_resource = true
			use_project_user_resource = true
			use_project_group_resource = true
			use_project_repository_resource = true
		}

		moved {
			from = project.{{ .name }}
			to   = project_project.{{ .name }}
		}
	`, params)

	// Provider factories are declared per step rather than on the TestCase,
	// because step one installs a released provider under the same name.
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(newResourceName, verifyProject),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"project": {
						Source:            "jfrog/project",
						VersionConstraint: "1.4.0",
					},
				},
				Config: schemaV3Config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(legacyResourceName, "key", params["project_key"].(string)),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Config:                   movedConfig,
				ExpectError:              regexp.MustCompile("Unable to Move Project State"),
			},
			{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Config:                   legacyCurrentConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(legacyResourceName, "key", params["project_key"].(string)),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Config:                   movedConfig,
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
