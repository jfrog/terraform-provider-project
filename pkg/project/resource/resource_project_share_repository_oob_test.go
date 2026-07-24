package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

// TestAccProjectShareRepository_outOfBandRemoval is a regression test for
// https://github.com/jfrog/terraform-provider-project/issues/227
//
// When the share is removed outside of Terraform, Read() must remove the
// resource from state (so the next plan cleanly recreates it) instead of
// setting the required attribute target_project_key to null, which produced
// invalid state and perpetual replace/destroy drift.
func TestAccProjectShareRepository_outOfBandRemoval(t *testing.T) {
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

	_, fqrn, resourceName := testutil.MkNames("test-project-share-repo-oob", "project_share_repository")

	params := map[string]string{
		"project_name":  projectName,
		"project_key":   projectKey,
		"repo_key":      repoKey,
		"resource_name": resourceName,
	}

	temp := `
		resource "artifactory_local_generic_repository" "{{ .repo_key }}" {
			key = "{{ .repo_key }}"

			lifecycle {
				ignore_changes = ["project_key", "project_environments"]
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

		resource "project_share_repository" "{{ .resource_name }}" {
			repo_key           = artifactory_local_generic_repository.{{ .repo_key }}.key
			target_project_key = project.{{ .project_name }}.key
			read_only          = true
		}
	`

	config := util.ExecuteTemplate("TestAccProjectShareRepositoryOOB", temp, params)

	// Deletes the share directly via the Access API, simulating an
	// out-of-band change (e.g. another user, Crossplane reconcile, UI).
	unshareOutOfBand := func() {
		resp, err := client.R().
			SetPathParams(map[string]string{
				"repo_key":           repoKey,
				"target_project_key": projectKey,
			}).
			SetQueryParam("readOnly", "true").
			Delete("access/api/v1/projects/_/share/repositories/{repo_key}/{target_project_key}")
		if err != nil {
			t.Fatalf("failed to unshare repo out-of-band: %v", err)
		}
		if resp.IsError() {
			t.Fatalf("failed to unshare repo out-of-band: %s", resp.String())
		}
	}

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
					resource.TestCheckResourceAttr(fqrn, "target_project_key", params["project_key"]),
					resource.TestCheckResourceAttr(fqrn, "read_only", "true"),
				),
			},
			{
				// Remove the share out-of-band, then run a refresh-only pass.
				// Read() must drop the resource from state, so the plan is
				// non-empty (Terraform wants to recreate it). Under the bug,
				// Read() instead wrote target_project_key = null, keeping an
				// invalid resource in state.
				PreConfig:          unshareOutOfBand,
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check: func(s *terraform.State) error {
					// The fix removes the resource from state. The bug kept it
					// in state with target_project_key = null, so asserting
					// absence is what distinguishes the two behaviors.
					if _, ok := s.RootModule().Resources[fqrn]; ok {
						return fmt.Errorf("expected %s to be removed from state after refresh (share deleted out-of-band), but it is still present", fqrn)
					}
					return nil
				},
			},
			{
				// The resource must cleanly recreate rather than being stuck
				// in a perpetual replace/destroy loop with a null required
				// attribute.
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "repo_key", params["repo_key"]),
					resource.TestCheckResourceAttr(fqrn, "target_project_key", params["project_key"]),
					resource.TestCheckResourceAttr(fqrn, "read_only", "true"),
				),
			},
		},
	})
}
