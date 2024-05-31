package project_test

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	acctest "github.com/jfrog/terraform-provider-project/pkg/project/acctest"
	project "github.com/jfrog/terraform-provider-project/pkg/project/resource"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
	"golang.org/x/exp/slices"
)

func TestAccProjectEnvironment_UpgradeFromSDKv2(t *testing.T) {
	_, _, projectName := testutil.MkNames("test-project-", "project")
	_, fqrn, projectEnvironmentName := testutil.MkNames("test-env-", "project_environment")

	projectKey := fmt.Sprintf("test%s", strings.ToLower(acctest.RandSeq(6)))

	params := map[string]any{
		"name":         projectEnvironmentName,
		"project_name": projectName,
		"project_key":  projectKey,
	}

	template := `
		resource "project" "{{ .project_name }}" {
			key          = "{{ .project_key }}"
			display_name = "{{ .project_key }}"
			admin_privileges {
				manage_members   = true
				manage_resources = true
				index_resources  = true
			}
		}

		resource "project_environment" "{{ .name }}" {
			name        = "{{ .name }}"
			project_key = project.{{ .project_name }}.key
		}
	`

	config := util.ExecuteTemplate("TestAccProjectEnvironment", template, params)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"project": {
						Source:            "jfrog/project",
						VersionConstraint: "1.6.0",
					},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "name", params["name"].(string)),
					resource.TestCheckResourceAttr(fqrn, "project_key", params["project_key"].(string)),
				),
			},
			{
				ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
				ConfigPlanChecks:         testutil.ConfigPlanChecks(fqrn),
			},
		},
	})
}

func TestAccProjectEnvironment_full(t *testing.T) {
	name := strings.ToLower(acctest.RandSeq(10))
	projectKey := strings.ToLower(acctest.RandSeq(10))
	resourceName := fmt.Sprintf("project_environment.%s", name)

	params := map[string]any{
		"env_id":      name,
		"name":        name,
		"project_key": projectKey,
	}

	template := `
		resource "project" "{{ .project_key }}" {
			key          = "{{ .project_key }}"
			display_name = "{{ .project_key }}"
			admin_privileges {
				manage_members   = true
				manage_resources = true
				index_resources  = true
			}
		}

		resource "project_environment" "{{ .env_id }}" {
			name        = "{{ .name }}"
			project_key = project.{{ .project_key }}.key
		}
	`

	enviroment := util.ExecuteTemplate("TestAccProjectEnvironment", template, params)

	updateParams := map[string]any{
		"env_id":      name,
		"name":        strings.ToLower(acctest.RandSeq(10)),
		"project_key": projectKey,
	}

	enviromentUpdated := util.ExecuteTemplate("TestAccProjectEnvironment", template, updateParams)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyEnvironment(projectKey, id, request)
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: enviroment,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", fmt.Sprintf("%s-%s", projectKey, params["name"])),
					resource.TestCheckResourceAttr(resourceName, "name", params["name"].(string)),
					resource.TestCheckResourceAttr(resourceName, "project_key", params["project_key"].(string)),
				),
			},
			{
				Config: enviromentUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", fmt.Sprintf("%s-%s", projectKey, updateParams["name"])),
					resource.TestCheckResourceAttr(resourceName, "name", updateParams["name"].(string)),
					resource.TestCheckResourceAttr(resourceName, "project_key", updateParams["project_key"].(string)),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportState:                          true,
				ImportStateId:                        fmt.Sprintf("%s:%s", updateParams["project_key"], updateParams["name"]),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccProjectEnvironment_invalid_length(t *testing.T) {
	name := fmt.Sprintf("env%s", strings.ToLower(acctest.RandSeq(15)))
	projectKey := fmt.Sprintf("project%s", strings.ToLower(acctest.RandSeq(7)))
	resourceName := fmt.Sprintf("project_environment.%s", name)

	params := map[string]any{
		"name":        name,
		"project_key": projectKey,
	}

	template := `
		resource "project" "{{ .project_key }}" {
			key          = "{{ .project_key }}"
			display_name = "{{ .project_key }}"
			admin_privileges {
				manage_members   = true
				manage_resources = true
				index_resources  = true
			}
		}

		resource "project_environment" "{{ .name }}" {
			name        = "{{ .name }}"
			project_key = project.{{ .project_key }}.key
		}
	`

	enviroment := util.ExecuteTemplate("TestAccProjectEnvironment", template, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { acctest.PreCheck(t) },
		CheckDestroy: acctest.VerifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			return verifyEnvironment(projectKey, id, request)
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      enviroment,
				ExpectError: regexp.MustCompile(`.*Combined length of project_key and name \(separated by '-'\) cannot exceed 32.*`),
			},
		},
	})
}

func verifyEnvironment(projectKey, id string, request *resty.Request) (*resty.Response, error) {
	var envs []project.ProjectEnvironmentAPIModel

	resp, err := request.
		SetPathParam("projectKey", projectKey).
		SetResult(&envs).
		Get(project.ProjectEnvironmentUrl)
	if err != nil {
		return resp, err
	}
	if resp.IsError() && resp.StatusCode() != http.StatusNotFound {
		return resp, fmt.Errorf("%s", resp.String())
	}

	if len(envs) > 0 {
		return resp, nil
	}

	envExists := slices.ContainsFunc(envs, func(e project.ProjectEnvironmentAPIModel) bool {
		return e.Name == fmt.Sprintf("%s-%s", projectKey, id)
	})

	if envExists {
		return resp, fmt.Errorf("environment %s still exist", id)
	}

	return resp, nil
}
