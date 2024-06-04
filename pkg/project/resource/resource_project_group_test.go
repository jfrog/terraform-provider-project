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
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccProjectGroup_UpgradeFromSDKv2(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, groupName := testutil.MkNames("test-project-group-", "project_group")

	projectKey := strings.ToLower(acctest.RandSeq(10))

	params := map[string]string{
		"project_name": projectName,
		"project_key":  projectKey,
		"group":        groupName,
		"roles":        `["Developer","Project Admin"]`,
	}

	template := `
		resource "artifactory_group" "{{ .group }}" {
			name = "{{ .group }}"
		}

		resource "project" "{{ .project_name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
		}

		resource "project_group" "{{ .group }}" {
			project_key = project.{{ .project_name }}.key
			name = artifactory_group.{{ .group }}.name
			roles = {{ .roles }}
		}
	`

	config := util.ExecuteTemplate("TestAccProjectGroup", template, params)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"artifactory": {
						Source: "jfrog/artifactory",
					},
					"project": {
						Source:            "jfrog/project",
						VersionConstraint: "1.6.0",
					},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "project_key", params["project_key"]),
					resource.TestCheckResourceAttr(fqrn, "name", groupName),
					resource.TestCheckResourceAttr(fqrn, "roles.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "roles.0", "Developer"),
					resource.TestCheckResourceAttr(fqrn, "roles.1", "Project Admin"),
				),
			},
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"artifactory": {
						Source: "jfrog/artifactory",
					},
				},
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
				ConfigPlanChecks:         testutil.ConfigPlanChecks(fqrn),
			},
		},
	})
}

func TestAccProjectGroup_full(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, groupName := testutil.MkNames("test-project-group-", "project_group")

	projectKey := strings.ToLower(acctest.RandSeq(10))

	params := map[string]string{
		"project_name": projectName,
		"project_key":  projectKey,
		"group":        groupName,
		"roles":        `["Developer", "Project Admin"]`,
	}

	template := `
		resource "artifactory_group" "{{ .group }}" {
			name = "{{ .group }}"
		}

		resource "project" "{{ .project_name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
		}

		resource "project_group" "{{ .group }}" {
			project_key = project.{{ .project_name }}.key
			name = artifactory_group.{{ .group }}.name
			roles = {{ .roles }}
		}
	`

	config := util.ExecuteTemplate("TestAccProjectGroup", template, params)

	updateParams := map[string]string{
		"project_name": params["project_name"],
		"project_key":  params["project_key"],
		"group":        params["group"],
		"roles":        `["Developer"]`,
	}

	configUpdated := util.ExecuteTemplate("TestAccProjectGroup", template, updateParams)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(fqrn, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyProjectGroup(groupName, projectKey, request)
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "project_key", params["project_key"]),
					resource.TestCheckResourceAttr(fqrn, "name", groupName),
					resource.TestCheckResourceAttr(fqrn, "roles.#", "2"),
					resource.TestCheckResourceAttr(fqrn, "roles.0", "Developer"),
					resource.TestCheckResourceAttr(fqrn, "roles.1", "Project Admin"),
				),
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "project_key", params["project_key"]),
					resource.TestCheckResourceAttr(fqrn, "name", groupName),
					resource.TestCheckResourceAttr(fqrn, "roles.#", "1"),
					resource.TestCheckResourceAttr(fqrn, "roles.0", "Developer"),
				),
			},
			{
				ResourceName:      fqrn,
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s:%s", updateParams["project_key"], updateParams["group"]),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccProjectGroup_invalid_roles(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, groupName := testutil.MkNames("test-project-group-", "project_group")

	projectKey := strings.ToLower(acctest.RandSeq(10))

	params := map[string]string{
		"project_name": projectName,
		"project_key":  projectKey,
		"group":        groupName,
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
		}

		resource "project_group" "{{ .group }}" {
			project_key = project.{{ .project_name }}.key
			name = artifactory_group.{{ .group }}.name
			roles = []
		}
	`

	config := util.ExecuteTemplate("TestAccProjectGroup", template, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(fqrn, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyProjectGroup(groupName, projectKey, request)
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source: "jfrog/artifactory",
			},
		},
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`.*Attribute roles set must contain at least 1 elements, got: 0.*`),
			},
		},
	})
}

func verifyProjectGroup(name, projectKey string, request *resty.Request) (*resty.Response, error) {
	return request.
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"name":       name,
		}).
		Get(project.ProjectGroupsUrl)
}
