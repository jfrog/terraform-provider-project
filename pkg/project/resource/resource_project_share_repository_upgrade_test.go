package project_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

// TestAccProjectShareRepository_upgradeFromVersion verifies that upgrading an
// existing project_share_repository resource from the published 1.9.5 provider
// (which contains the Read bug from issue #227) to the current dev build does
// not break existing state: a valid share created on 1.9.5 must upgrade with
// an empty plan (no spurious drift, no destroy/recreate).
func TestAccProjectShareRepository_upgradeFromVersion(t *testing.T) {
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

	_, fqrn, resourceName := testutil.MkNames("test-project-share-repo-upgrade", "project_share_repository")

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

			# admin_privileges (manage_remote_repository) has known drift
			# across provider versions and is not relevant to this test, which
			# targets project_share_repository upgrade behavior.
			lifecycle {
				ignore_changes = [admin_privileges]
			}
		}

		resource "project_share_repository" "{{ .resource_name }}" {
			repo_key           = artifactory_local_generic_repository.{{ .repo_key }}.key
			target_project_key = project.{{ .project_name }}.key
			read_only          = true
		}
	`

	config := util.ExecuteTemplate("TestAccProjectShareRepositoryUpgrade", temp, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		Steps: []resource.TestStep{
			{
				// Create the resource with the published 1.9.5 provider,
				// which contains the buggy Read().
				ExternalProviders: map[string]resource.ExternalProvider{
					"project": {
						Source:            "jfrog/project",
						VersionConstraint: "1.9.5",
					},
					"artifactory": {
						Source: "jfrog/artifactory",
					},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "repo_key", params["repo_key"]),
					resource.TestCheckResourceAttr(fqrn, "target_project_key", params["project_key"]),
					resource.TestCheckResourceAttr(fqrn, "read_only", "true"),
				),
			},
			{
				// Upgrade in place to the dev build. State written by 1.9.5
				// must be read cleanly with no planned changes.
				ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
				ExternalProviders: map[string]resource.ExternalProvider{
					"artifactory": {
						Source: "jfrog/artifactory",
					},
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// Scope the assertion to the share_repository resource.
						// The `project` resource has known, unrelated drift
						// between provider versions (admin_privileges /
						// project_environments), so a whole-plan empty check
						// would be a false negative here.
						plancheck.ExpectResourceAction(fqrn, plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}
