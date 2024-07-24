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

func TestAccProjectShareRepository_full(t *testing.T) {
	client := acctest.GetTestResty(t)
	version, err := util.GetArtifactoryVersion(client)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := util.CheckVersion(version, "7.90.1")
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Skipf("Artifactory version %s is earlier than 7.90.1", version)
	}

	projectKey1 := strings.ToLower(acctest.RandSeq(10))
	projectKey2 := strings.ToLower(acctest.RandSeq(10))
	projectName1 := fmt.Sprintf("tftestprojects%s", projectKey1)
	projectName2 := fmt.Sprintf("tftestprojects%s", projectKey2)

	repoKey := fmt.Sprintf("repo%d", testutil.RandomInt())

	_, fqrn, resourceName := testutil.MkNames("test-project-share-repo", "project_share_repository")

	params := map[string]string{
		"project_name_1": projectName1,
		"project_key_1":  projectKey1,
		"repo_key":       repoKey,
		"resource_name":  resourceName,
	}

	temp := `
		resource "artifactory_local_generic_repository" "{{ .repo_key }}" {
			key = "{{ .repo_key }}"

			lifecycle {
				ignore_changes = ["project_key"]
			}
		}

		resource "project" "{{ .project_name_1 }}" {
			key          = "{{ .project_key_1 }}"
			display_name = "{{ .project_name_1 }}"
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

		resource "project_share_repository" "{{ .resource_name }}" {
			repo_key = artifactory_local_generic_repository.{{ .repo_key }}.key
			target_project_key = project.{{ .project_name_1 }}.key
		}
	`

	config := util.ExecuteTemplate("TestAccProjectShareRepository", temp, params)

	updatedTemp := `
		resource "artifactory_local_generic_repository" "{{ .repo_key }}" {
			key = "{{ .repo_key }}"

			lifecycle {
				ignore_changes = ["project_key"]
			}
		}

		resource "project" "{{ .project_name_2 }}" {
			key          = "{{ .project_key_2 }}"
			display_name = "{{ .project_name_2 }}"
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

		resource "project_share_repository" "{{ .resource_name }}" {
			repo_key = artifactory_local_generic_repository.{{ .repo_key }}.key
			target_project_key = project.{{ .project_name_2 }}.key
		}
	`
	updateParams := map[string]string{
		"project_name_2": projectName2,
		"project_key_2":  projectKey2,
		"repo_key":       params["repo_key"],
		"resource_name":  resourceName,
	}

	configUpdated := util.ExecuteTemplate("TestAccProjectShareRepository", updatedTemp, updateParams)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
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
					resource.TestCheckResourceAttr(fqrn, "repo_key", params["repo_key"]),
					resource.TestCheckResourceAttr(fqrn, "target_project_key", params["project_key_1"]),
				),
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "repo_key", updateParams["repo_key"]),
					resource.TestCheckResourceAttr(fqrn, "target_project_key", updateParams["project_key_2"]),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportStateId:                        fmt.Sprintf("%s:%s", updateParams["repo_key"], updateParams["project_key_2"]),
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "repo_key",
			},
		},
	})
}
