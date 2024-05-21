package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccProject_membership(t *testing.T) {
	name := "tftestprojects" + acctest.RandSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(acctest.RandSeq(10))

	username1 := "user1"
	username2 := "user2"
	developeRole := "Developer"
	contributorRole := "Contributor"

	params := map[string]interface{}{
		"name":            name,
		"project_key":     projectKey,
		"username1":       username1,
		"username2":       username2,
		"developeRole":    developeRole,
		"contributorRole": contributorRole,
	}

	initialConfig := util.ExecuteTemplate("TestAccProjectMember", `
		resource "artifactory_managed_user" "{{ .username1 }}" {
			name = "{{ .username1 }}"
			email = "{{ .username1 }}@tempurl.org"
			password = "Password!123"
		}

		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_user_resource = false

			member {
				name = artifactory_managed_user.{{ .username1 }}.name
				roles = ["{{ .developeRole }}"]
			}
		}
	`, params)

	addMembersConfig := util.ExecuteTemplate("TestAccProjectMember", `
		resource "artifactory_managed_user" "{{ .username1 }}" {
			name = "{{ .username1 }}"
			email = "{{ .username1 }}@tempurl.org"
			password = "Password!123"
		}

		resource "artifactory_managed_user" "{{ .username2 }}" {
			name = "{{ .username2 }}"
			email = "{{ .username2 }}@tempurl.org"
			password = "Password!123"
		}

		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_user_resource = false

			member {
				name = artifactory_managed_user.{{ .username1 }}.name
				roles = ["{{ .developeRole }}", "{{ .contributorRole }}"]
			}

			member {
				name = artifactory_managed_user.{{ .username2 }}.name
				roles = ["{{ .contributorRole }}"]
			}
		}
	`, params)

	noMemberConfig := util.ExecuteTemplate("TestAccProjectMember", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_user_resource = false
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		CheckDestroy:             acctest.VerifyDeleted(resourceName, verifyProject),
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "member.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "member.0.name", username1),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.0", developeRole),
				),
			},
			{
				Config: addMembersConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "member.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "member.0.name", username1),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.0", contributorRole),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.1", developeRole),
					resource.TestCheckResourceAttr(resourceName, "member.1.name", username2),
					resource.TestCheckResourceAttr(resourceName, "member.1.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "member.1.roles.0", contributorRole),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"use_project_role_resource",
					"use_project_user_resource",
					"use_project_group_resource",
					"use_project_repository_resource",
				},
			},
			{
				Config: noMemberConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "member.#", "0"),
				),
			},
		},
	})
}

func TestAccProject_group(t *testing.T) {
	name := "tftestprojects" + acctest.RandSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(acctest.RandSeq(10))

	group1 := "group1"
	group2 := "group2"
	developeRole := "Developer"
	contributorRole := "Contributor"

	params := map[string]interface{}{
		"name":            name,
		"project_key":     projectKey,
		"group1":          group1,
		"group2":          group2,
		"developeRole":    developeRole,
		"contributorRole": contributorRole,
	}

	initialConfig := util.ExecuteTemplate("TestAccProjectGroup", `
		resource "artifactory_group" "{{ .group1 }}" {
			name = "{{ .group1 }}"
		}

		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_group_resource = false

			group {
				name = artifactory_group.{{ .group1 }}.name
				roles = ["{{ .developeRole }}"]
			}
		}
	`, params)

	addGroupConfig := util.ExecuteTemplate("TestAccProjectGroup", `
		resource "artifactory_group" "{{ .group1 }}" {
			name = "{{ .group1 }}"
		}

		resource "artifactory_group" "{{ .group2 }}" {
			name = "{{ .group2 }}"
		}

		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_group_resource = false

			group {
				name = artifactory_group.{{ .group1 }}.name
				roles = ["{{ .developeRole }}", "{{ .contributorRole }}"]
			}

			group {
				name = artifactory_group.{{ .group2 }}.name
				roles = ["{{ .contributorRole }}"]
			}
		}
	`, params)

	noGroupConfig := util.ExecuteTemplate("TestAccProjectGroup", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			use_project_group_resource = false
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },

		CheckDestroy:             acctest.VerifyDeleted(resourceName, verifyProject),
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "group.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "group.0.name", group1),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.0", developeRole),
				),
			},
			{
				Config: addGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "group.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "group.0.name", group1),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.0", contributorRole),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.1", developeRole),
					resource.TestCheckResourceAttr(resourceName, "group.1.name", group2),
					resource.TestCheckResourceAttr(resourceName, "group.1.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "group.1.roles.0", contributorRole),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"use_project_role_resource",
					"use_project_user_resource",
					"use_project_group_resource",
					"use_project_repository_resource",
				},
			},
			{
				Config: noGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "group.#", "0"),
				),
			},
		},
	})
}
