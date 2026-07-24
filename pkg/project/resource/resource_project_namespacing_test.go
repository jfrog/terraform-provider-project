package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	"github.com/jfrog/terraform-provider-shared/util"
)

// TestAccProject_namespacedName verifies Phase 1 of issue #210: the new
// `jfrog_project` resource name works end-to-end (create/read/update).
func TestAccProject_namespacedName(t *testing.T) {
	name := fmt.Sprintf("tftestprojects%s", acctest.RandSeq(10))
	resourceName := fmt.Sprintf("jfrog_project.%s", name)
	key := strings.ToLower(acctest.RandSeq(6))

	config := util.ExecuteTemplate("TestAccProjectNamespaced", `
		resource "jfrog_project" "{{ .name }}" {
			key          = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description  = "test description"
			admin_privileges {
				manage_members           = true
				manage_resources         = true
				manage_remote_repository = true
				index_resources          = true
			}
			max_storage_in_gibibytes   = 1
			block_deployments_on_limit = true
			email_notification         = false
		}
	`, map[string]interface{}{
		"name":        name,
		"project_key": key,
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             acctest.VerifyDeleted(resourceName, verifyProject),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", key),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     key,
				ImportStateVerify: true,
				// Config-only attributes that are not returned by the API on import.
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
