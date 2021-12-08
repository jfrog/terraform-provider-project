package projects

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccProjectRole(t *testing.T) {
	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(6))

	role1 := "role 1"
	role2 := "role 2"

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"role1":       role1,
		"role2":       role2,
	}

	initialConfig := executeTemplate("TestAccProjectRole", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			role {
				name = "{{ .role1 }}"
				description = "test description"
				type = "CUSTOM"
				environments = ["DEV"]
				actions = ["READ_REPOSITORY"]
			}
		}
	`, params)

	addUserConfig := executeTemplate("TestAccProjectRole", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			role {
				name = "{{ .role1 }}"
				description = "test description"
				type = "CUSTOM"
				environments = ["DEV", "PROD"]
				actions = ["READ_REPOSITORY"]
			}

			role {
				name = "{{ .role2 }}"
				description = "test description 2"
				type = "CUSTOM"
				environments = ["DEV", "PROD"]
				actions = ["READ_REPOSITORY", "ANNOTATE_REPOSITORY"]
			}
		}
	`, params)

	noUserConfig := executeTemplate("TestAccProjectRole", `
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
		},
		CheckDestroy:      verifyDeleted(resourceName, verifyProject),
		ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "role.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role.0.name", role1),
					resource.TestCheckResourceAttr(resourceName, "role.0.environments.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role.0.environments.0", "DEV"),
					resource.TestCheckResourceAttr(resourceName, "role.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role.0.actions.0", "READ_REPOSITORY"),
				),
			},
			{
				Config: addUserConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "role.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role.0.name", role2),
					resource.TestCheckResourceAttr(resourceName, "role.0.environments.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role.0.environments.0", "DEV"),
					resource.TestCheckResourceAttr(resourceName, "role.0.environments.1", "PROD"),
					resource.TestCheckResourceAttr(resourceName, "role.0.actions.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role.0.actions.0", "ANNOTATE_REPOSITORY"),
					resource.TestCheckResourceAttr(resourceName, "role.0.actions.1", "READ_REPOSITORY"),
					resource.TestCheckResourceAttr(resourceName, "role.1.name", role1),
					resource.TestCheckResourceAttr(resourceName, "role.1.environments.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "role.1.environments.0", "DEV"),
					resource.TestCheckResourceAttr(resourceName, "role.1.environments.1", "PROD"),
					resource.TestCheckResourceAttr(resourceName, "role.1.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role.1.actions.0", "READ_REPOSITORY"),
				),
			},
			{
				Config: noUserConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckNoResourceAttr(resourceName, "role"),
				),
			},
		},
	})
}
