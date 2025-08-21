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
				ignore_changes = ["project_key", "project_environments"]
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
			read_only = true
		}
	`

	config := util.ExecuteTemplate("TestAccProjectShareRepository", temp, params)

	updatedTemp := `
		resource "artifactory_local_generic_repository" "{{ .repo_key }}" {
			key = "{{ .repo_key }}"

			lifecycle {
				ignore_changes = ["project_key", "project_environments"]
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
					resource.TestCheckResourceAttr(fqrn, "read_only", "true"),
				),
			},
			{
				Config: configUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "repo_key", updateParams["repo_key"]),
					resource.TestCheckResourceAttr(fqrn, "target_project_key", updateParams["project_key_2"]),
					resource.TestCheckResourceAttr(fqrn, "read_only", "false"),
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

func TestAccProjectShareRepositoryWithMultipleProjects(t *testing.T) {
	// The goal of this test is to simulate the race condition, when the repository was only shared with one project (first in the list) if
	// the loop was used in "project_share_repository" resource
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
	projectKey3 := strings.ToLower(acctest.RandSeq(10))
	projectKey4 := strings.ToLower(acctest.RandSeq(10))
	projectName1 := fmt.Sprintf("tftestprojects1_%s", projectKey1)
	projectName2 := fmt.Sprintf("tftestprojects2_%s", projectKey2)
	projectName3 := fmt.Sprintf("tftestprojects3_%s", projectKey3)
	projectName4 := fmt.Sprintf("tftestprojects4_%s", projectKey4)

	repoKey := fmt.Sprintf("repo%d", testutil.RandomInt())

	fqrn := "project_share_repository.share_repo"

	params := map[string]string{
		"project_name_1": projectName1,
		"project_key_1":  projectKey1,
		"project_name_2": projectName2,
		"project_key_2":  projectKey2,
		"project_name_3": projectName3,
		"project_key_3":  projectKey3,
		"project_name_4": projectName4,
		"project_key_4":  projectKey4,
		"repo_key":       repoKey,
	}
	// Creating projects without for each loop, they are not supported in tf tests
	temp := `
		resource "artifactory_local_generic_repository" "repo" {
		  key          = "{{ .repo_key }}"
		  description  = "Lab repository for troubleshooting - {{ .repo_key }}"
			
          lifecycle {
				ignore_changes = ["project_key", "project_environments"]
			}
		}
		
		# Create 4 projects
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

		resource "project" "{{ .project_name_3 }}" {
			key          = "{{ .project_key_3 }}"
			display_name = "{{ .project_name_3 }}"
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

		resource "project" "{{ .project_name_4 }}" {
			key          = "{{ .project_key_4 }}"
			display_name = "{{ .project_name_4 }}"
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

		# Add repository to {{ .project_name_1 }}
		resource "project_repository" "add_to_project1" {
		  project_key = project.{{ .project_name_1 }}.key
		  key         = artifactory_local_generic_repository.repo.key
		}
		
		# Share ONE repo with the other three projects
		resource "project_share_repository" "share_repo" {
		  count = 3
		
		  repo_key = artifactory_local_generic_repository.repo.key
		  target_project_key = element(
			[
			  project.{{ .project_name_2 }}.key,
			  project.{{ .project_name_3 }}.key,
			  project.{{ .project_name_4 }}.key
			],
			count.index
		  )
		  read_only  = true
		  depends_on = [project_repository.add_to_project1]
		}
	`

	config := util.ExecuteTemplate("TestAccProjectShareRepository", temp, params)

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
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.0", fqrn), "repo_key", params["repo_key"]),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.0", fqrn), "target_project_key", params["project_key_2"]),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.0", fqrn), "read_only", "true"),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.1", fqrn), "repo_key", params["repo_key"]),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.1", fqrn), "target_project_key", params["project_key_3"]),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.1", fqrn), "read_only", "true"),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.2", fqrn), "repo_key", params["repo_key"]),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.2", fqrn), "target_project_key", params["project_key_4"]),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.2", fqrn), "read_only", "true"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("%s[0]", fqrn),
				ImportStateId:     fmt.Sprintf("%s:%s", params["repo_key"], params["project_key_2"]),
				ImportState:       true,
				ImportStateVerify: false,
			},
		},
	})
}
