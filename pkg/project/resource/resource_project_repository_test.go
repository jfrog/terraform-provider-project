package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

func TestAccProjectRepository_UpgradeFromSDKv2(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, projectRepoName := testutil.MkNames("test-project-repo-", "project_repository")

	projectKey := strings.ToLower(acctest.RandSeq(10))
	repoKey1 := fmt.Sprintf("repo%d", testutil.RandomInt())
	repoKey2 := fmt.Sprintf("repo%d", testutil.RandomInt())

	params := map[string]interface{}{
		"project_name":      projectName,
		"project_key":       projectKey,
		"repo_key":          repoKey1,
		"repo_key_1":        repoKey1,
		"repo_key_2":        repoKey2,
		"project_repo_name": projectRepoName,
	}

	template := `
		resource "artifactory_local_generic_repository" "{{ .repo_key_1 }}" {
			key = "{{ .repo_key_1 }}"

			lifecycle {
				ignore_changes = ["project_key"]
			}
		}

		resource "artifactory_local_generic_repository" "{{ .repo_key_2 }}" {
			key = "{{ .repo_key_2 }}"

			lifecycle {
				ignore_changes = ["project_key"]
			}
		}

		resource "project" "{{ .project_name }}" {
			key          = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			description  = "test description"
			admin_privileges {
				manage_members   = true
				manage_resources = true
				index_resources  = true
			}
			max_storage_in_gibibytes   = 1
			block_deployments_on_limit = true
			email_notification         = false
		}

		resource "project_repository" "{{ .project_repo_name }}" {
			project_key = project.{{ .project_name }}.key
			key         = artifactory_local_generic_repository.{{ .repo_key }}.key
		}
	`

	config := util.ExecuteTemplate("TestAccProjectRepository", template, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"artifactory": {
						Source:            "jfrog/artifactory",
						VersionConstraint: "10.3.3",
					},
					"project": {
						Source:            "jfrog/project",
						VersionConstraint: "1.6.1",
					},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "project_key", params["project_key"].(string)),
					resource.TestCheckResourceAttr(fqrn, "key", params["repo_key"].(string)),
				),
			},
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"artifactory": {
						Source:            "jfrog/artifactory",
						VersionConstraint: "10.3.3",
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

func TestAccProjectRepository_full(t *testing.T) {
	projectKey := strings.ToLower(acctest.RandSeq(10))
	projectName := fmt.Sprintf("tftestprojects%s", projectKey)

	repoKey1 := fmt.Sprintf("repo%d", testutil.RandomInt())
	repoKey2 := fmt.Sprintf("repo%d", testutil.RandomInt())

	resourceName1 := fmt.Sprintf("project_repository.%s-%s", projectKey, repoKey1)
	resourceName2 := fmt.Sprintf("project_repository.%s-%s", projectKey, repoKey2)

	params := map[string]interface{}{
		"project_name": projectName,
		"project_key":  projectKey,
		"repo_key":     repoKey1,
		"repo_key_1":   repoKey1,
		"repo_key_2":   repoKey2,
	}

	template := `
		resource "artifactory_local_generic_repository" "{{ .repo_key_1 }}" {
			key = "{{ .repo_key_1 }}"

			lifecycle {
				ignore_changes = ["project_key"]
			}
		}

		resource "artifactory_local_generic_repository" "{{ .repo_key_2 }}" {
			key = "{{ .repo_key_2 }}"

			lifecycle {
				ignore_changes = ["project_key"]
			}
		}

		resource "project" "{{ .project_name }}" {
			key          = "{{ .project_key }}"
			display_name = "{{ .project_name }}"
			description  = "test description"
			admin_privileges {
				manage_members   = true
				manage_resources = true
				index_resources  = true
			}
			max_storage_in_gibibytes   = 1
			block_deployments_on_limit = true
			email_notification         = false
		}

		resource "project_repository" "{{ .project_key }}-{{ .repo_key }}" {
			project_key = project.{{ .project_name }}.key
			key         = artifactory_local_generic_repository.{{ .repo_key }}.key
		}
	`

	config := util.ExecuteTemplate("TestAccProjectRepository", template, params)

	updateParams := map[string]interface{}{
		"project_name": params["project_name"],
		"project_key":  params["project_key"],
		"repo_key":     params["repo_key_2"],
		"repo_key_1":   params["repo_key_1"],
		"repo_key_2":   params["repo_key_2"],
	}

	configUpdated := util.ExecuteTemplate("TestAccProjectRepository", template, updateParams)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"artifactory": {
				Source:            "jfrog/artifactory",
				VersionConstraint: "10.3.3",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName1, "project_key", params["project_key"].(string)),
					resource.TestCheckResourceAttr(resourceName1, "key", params["repo_key"].(string)),
				),
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName2, "project_key", updateParams["project_key"].(string)),
					resource.TestCheckResourceAttr(resourceName2, "key", updateParams["repo_key"].(string)),
				),
			},
			{
				ResourceName:      resourceName2,
				ImportStateId:     fmt.Sprintf("%s:%s", projectKey, updateParams["repo_key"]),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
