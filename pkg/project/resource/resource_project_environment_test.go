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
	"github.com/jfrog/terraform-provider-shared/util"
	"golang.org/x/exp/slices"
)

func TestAccProjectEnvironment(t *testing.T) {
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
			resp, err := verifyEnvironment(projectKey, id, request)
			return resp, err
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: enviroment,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", params["name"].(string)),
					resource.TestCheckResourceAttr(resourceName, "project_key", params["project_key"].(string)),
				),
			},
			{
				Config: enviromentUpdated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updateParams["name"].(string)),
					resource.TestCheckResourceAttr(resourceName, "project_key", updateParams["project_key"].(string)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateId:     fmt.Sprintf("%s:%s", projectKey, updateParams["name"]),
				ImportState:       true,
				ImportStateVerify: true,
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
			resp, err := verifyEnvironment(projectKey, id, request)
			return resp, err
		}),
		ProtoV6ProviderFactories: acctest.ProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      enviroment,
				ExpectError: regexp.MustCompile(`.*combined length of project_key and name \(separated by '-'\) cannot exceed 32 characters.*`),
			},
		},
	})
}

func verifyEnvironment(projectKey, id string, request *resty.Request) (*resty.Response, error) {
	envs := []project.ProjectEnvironment{}

	resp, err := request.
		SetPathParam("projectKey", projectKey).
		SetResult(&envs).
		Get(project.ProjectEnvironmentUrl)
	if err != nil {
		return resp, err
	}

	envExists := slices.ContainsFunc(envs, func(e project.ProjectEnvironment) bool {
		return e.Name == fmt.Sprintf("%s-%s", projectKey, id)
	})

	if envExists {
		return resp, fmt.Errorf("environment %s still exist", id)
	}

	return resp, nil
}
