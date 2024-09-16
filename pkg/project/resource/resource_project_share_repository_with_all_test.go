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

func TestAccProjectShareWithAllRepository_full(t *testing.T) {
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

	projectKey := strings.ToLower(acctest.RandSeq(10))
	projectName := fmt.Sprintf("tftestprojects%s", projectKey)

	repoKey := fmt.Sprintf("repo%d", testutil.RandomInt())

	_, fqrn, resourceName := testutil.MkNames("test-project-share-repo", "project_share_repository_with_all")

	params := map[string]interface{}{
		"project_name":   projectName,
		"project_key":    projectKey,
		"repo_key":       repoKey,
		"resource_name":  resourceName,
		"share_with_all": "true",
	}

	temp := `
		resource "artifactory_local_generic_repository" "{{ .repo_key }}" {
			key = "{{ .repo_key }}"

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

		resource "project_share_repository_with_all" "{{ .resource_name }}" {
			repo_key = artifactory_local_generic_repository.{{ .repo_key }}.key
			read_only = true

			depends_on = [
				project.{{ .project_name }}
			]
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

		resource "project_share_repository_with_all" "{{ .resource_name }}" {
			repo_key = artifactory_local_generic_repository.{{ .repo_key }}.key

			depends_on = [
				project.{{ .project_name }}
			]
		}
	`

	updatedConfig := util.ExecuteTemplate("TestAccProjectShareRepository", updatedTemp, params)

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
					resource.TestCheckResourceAttr(fqrn, "repo_key", params["repo_key"].(string)),
					resource.TestCheckResourceAttr(fqrn, "read_only", "true"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "repo_key", params["repo_key"].(string)),
					resource.TestCheckResourceAttr(fqrn, "read_only", "false"),
				),
			},
			{
				ResourceName:                         fqrn,
				ImportStateId:                        params["repo_key"].(string),
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "repo_key",
			},
		},
	})
}
