package projects

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func makeInvalidProjectKeyTestCase(invalidProjectKey string, t *testing.T) (*testing.T, resource.TestCase) {
	name := fmt.Sprintf("tftestprojects%s", randSeq(10))
	resourceName := fmt.Sprintf("project_project.%s", name)

	params := map[string]interface{}{
		"max_storage_in_gigabytes":   rand.Intn(100),
		"block_deployments_on_limit": randBool(),
		"email_notification":         randBool(),
		"manage_members":             randBool(),
		"manage_resources":           randBool(),
		"index_resources":            randBool(),
		"name":                       name,
		"project_key":                invalidProjectKey, //strings.ToLower(randSeq(20)),
	}
	project := executeTemplate("TestAccProjects", `
		resource "project_project" "{{ .name }}" {
            key = "{{ .project_key }}"
            display_name = "{{ .name }}"
            description = "test description"
            admin_privileges {
                manage_members = {{ .manage_members }}
                manage_resources = {{ .manage_resources }}
                index_resources = {{ .index_resources }}
            }
            max_storage_in_gigabytes = {{ .max_storage_in_gigabytes }}
			block_deployments_on_limit = {{ .block_deployments_on_limit }}
            email_notification = {{ .email_notification }}
        }
	`, params)

	return t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      verifyDeleted(resourceName, verifyProject),
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      project,
				ExpectError: regexp.MustCompile(`.*key must be 3 - 6 lowercase alphanumeric characters.*`),
			},
		},
	}
}

type testCase struct {
	Name  string
	Value string
}

func TestAccProjectsInvalidProjectKey(t *testing.T) {
	invalidProjectKeys := []testCase{
		testCase{
			Name:  "TooShort",
			Value: strings.ToLower(randSeq(2)),
		},
		testCase{
			Name:  "TooLong",
			Value: strings.ToLower(randSeq(7)),
		},
		testCase{
			Name:  "HasUppercase",
			Value: randSeq(8),
		},
	}

	for _, invalidProjectKey := range invalidProjectKeys {
		t.Run(fmt.Sprintf("TestProjectKey%s", invalidProjectKey.Name), func(t *testing.T) {
			resource.Test(makeInvalidProjectKeyTestCase(invalidProjectKey.Value, t))
		})
	}
}

func TestAccProjectInvalidDisplayName(t *testing.T) {
	name := fmt.Sprintf("tftestprojects%s", randSeq(20))
	resourceName := fmt.Sprintf("project_project.%s", name)

	params := map[string]interface{}{
		"max_storage_in_gigabytes":   rand.Intn(100),
		"block_deployments_on_limit": randBool(),
		"email_notification":         randBool(),
		"manage_members":             randBool(),
		"manage_resources":           randBool(),
		"index_resources":            randBool(),
		"name":                       name,
		"project_key":                strings.ToLower(randSeq(6)),
	}
	project := executeTemplate("TestAccProjects", `
		resource "project_project" "{{ .name }}" {
            key = "{{ .project_key }}"
            display_name = "{{ .name }}"
            description = "test description"
            admin_privileges {
                manage_members = {{ .manage_members }}
                manage_resources = {{ .manage_resources }}
                index_resources = {{ .index_resources }}
            }
            max_storage_in_gigabytes = {{ .max_storage_in_gigabytes }}
			block_deployments_on_limit = {{ .block_deployments_on_limit }}
            email_notification = {{ .email_notification }}
        }
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      verifyDeleted(resourceName, verifyProject),
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: project,
				ExpectError: regexp.MustCompile(`.*string must be less than or equal 32 characters long.*`),
			},
		},
	})
}

func TestAccProject(t *testing.T) {
	name := fmt.Sprintf("tftestprojects%s", randSeq(10))
	resourceName := fmt.Sprintf("project_project.%s", name)

	params := map[string]interface{}{
		"max_storage_in_gigabytes":   rand.Intn(100),
		"block_deployments_on_limit": randBool(),
		"email_notification":         randBool(),
		"manage_members":             randBool(),
		"manage_resources":           randBool(),
		"index_resources":            randBool(),
		"name":                       name,
		"project_key":                strings.ToLower(randSeq(6)),
	}
	project := executeTemplate("TestAccProjects", `
		resource "project_project" "{{ .name }}" {
            key = "{{ .project_key }}"
            display_name = "{{ .name }}"
            description = "test description"
            admin_privileges {
                manage_members = {{ .manage_members }}
                manage_resources = {{ .manage_resources }}
                index_resources = {{ .index_resources }}
            }
            max_storage_in_gigabytes = {{ .max_storage_in_gigabytes }}
			block_deployments_on_limit = {{ .block_deployments_on_limit }}
            email_notification = {{ .email_notification }}
        }
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      verifyDeleted(resourceName, verifyProject),
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: project,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "max_storage_in_gigabytes", fmt.Sprintf("%d", params["max_storage_in_gigabytes"])),
					resource.TestCheckResourceAttr(resourceName, "block_deployments_on_limit", fmt.Sprintf("%t", params["block_deployments_on_limit"])),
					resource.TestCheckResourceAttr(resourceName, "email_notification", fmt.Sprintf("%t", params["email_notification"])),
					resource.TestCheckResourceAttr(resourceName, "admin_privileges.0.manage_members", fmt.Sprintf("%t", params["manage_members"])),
					resource.TestCheckResourceAttr(resourceName, "admin_privileges.0.manage_resources", fmt.Sprintf("%t", params["manage_resources"])),
					resource.TestCheckResourceAttr(resourceName, "admin_privileges.0.index_resources", fmt.Sprintf("%t", params["index_resources"])),
				),
			},
		},
	})
}
