package project

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/jfrog/terraform-provider-shared/test"
)

func makeInvalidProjectKeyTestCase(invalidProjectKey string, t *testing.T) (*testing.T, resource.TestCase) {
	name := fmt.Sprintf("tftestprojects%s", randSeq(10))
	resourceName := fmt.Sprintf("project.%s", name)

	params := map[string]interface{}{
		"max_storage_in_gibibytes":   rand.Intn(100),
		"block_deployments_on_limit": test.RandBool(),
		"email_notification":         test.RandBool(),
		"manage_members":             test.RandBool(),
		"manage_resources":           test.RandBool(),
		"index_resources":            test.RandBool(),
		"name":                       name,
		"project_key":                invalidProjectKey, //strings.ToLower(randSeq(20)),
	}
	project := test.ExecuteTemplate("TestAccProjects", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = {{ .manage_members }}
				manage_resources = {{ .manage_resources }}
				index_resources = {{ .index_resources }}
			}
			max_storage_in_gibibytes = {{ .max_storage_in_gibibytes }}
			block_deployments_on_limit = {{ .block_deployments_on_limit }}
			email_notification = {{ .email_notification }}
		}
	`, params)

	return t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      verifyDeleted(resourceName, verifyProject),
		ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      project,
				ExpectError: regexp.MustCompile(`.*key must be 3 - 10 lowercase alphanumeric characters.*`),
			},
		},
	}
}

type testCase struct {
	Name  string
	Value string
}

func TestAccProjectInvalidProjectKey(t *testing.T) {
	invalidProjectKeys := []testCase{
		{
			Name:  "TooShort",
			Value: strings.ToLower(randSeq(2)),
		},
		{
			Name:  "TooLong",
			Value: strings.ToLower(randSeq(11)),
		},
		{
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

func testProjectConfig(name, key string) string {
	params := map[string]interface{}{
		"max_storage_in_gibibytes":   rand.Intn(100),
		"block_deployments_on_limit": test.RandBool(),
		"email_notification":         test.RandBool(),
		"manage_members":             test.RandBool(),
		"manage_resources":           test.RandBool(),
		"index_resources":            test.RandBool(),
		"name":                       name,
		"project_key":                key,
	}
	return test.ExecuteTemplate("TestAccProjects", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = {{ .manage_members }}
				manage_resources = {{ .manage_resources }}
				index_resources = {{ .index_resources }}
			}
			max_storage_in_gibibytes = {{ .max_storage_in_gibibytes }}
			block_deployments_on_limit = {{ .block_deployments_on_limit }}
			email_notification = {{ .email_notification }}
		}
	`, params)
}

func TestAccProjectInvalidDisplayName(t *testing.T) {
	name := fmt.Sprintf("invalidtestprojects%s", randSeq(20))
	resourceName := fmt.Sprintf("project.%s", name)
	project := testProjectConfig(name, strings.ToLower(randSeq(6)))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      verifyDeleted(resourceName, verifyProject),
		ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:      project,
				ExpectError: regexp.MustCompile(`.*string must be less than or equal 32 characters long.*`),
			},
		},
	})
}

func TestAccProjectUpdateKey(t *testing.T) {
	name := fmt.Sprintf("testprojects%s", randSeq(20))
	resourceName := fmt.Sprintf("project.%s", name)
	key1 := strings.ToLower(randSeq(6))
	config := testProjectConfig(name, key1)

	key2 := strings.ToLower(randSeq(6))
	configWithNewKey := testProjectConfig(name, key2)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      verifyDeleted(resourceName, verifyProject),
		ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", key1),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
				),
			},
			{
				Config: configWithNewKey,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", key2),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
				),
			},
		},
	})
}

func TestAccProject(t *testing.T) {
	name := fmt.Sprintf("tftestprojects%s", randSeq(10))
	resourceName := fmt.Sprintf("project.%s", name)

	username1 := "user1"
	email1 := username1 + "@tempurl.org"
	username2 := "user2"
	email2 := username2 + "@tempurl.org"
	group1 := "group1"
	group2 := "group2"
	repo1 := fmt.Sprintf("repo%d", test.RandomInt())
	repo2 := fmt.Sprintf("repo%d", test.RandomInt())

	params := map[string]interface{}{
		"max_storage_in_gibibytes":   rand.Intn(100),
		"block_deployments_on_limit": test.RandBool(),
		"email_notification":         test.RandBool(),
		"manage_members":             test.RandBool(),
		"manage_resources":           test.RandBool(),
		"index_resources":            test.RandBool(),
		"name":                       name,
		"project_key":                strings.ToLower(randSeq(6)),
		"username1":                  username1,
		"username2":                  username2,
		"group1":                     group1,
		"group2":                     group2,
		"repo1":                      repo1,
		"repo2":                      repo2,
	}

	project := test.ExecuteTemplate("TestAccProjects", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = {{ .manage_members }}
				manage_resources = {{ .manage_resources }}
				index_resources = {{ .index_resources }}
			}
			max_storage_in_gibibytes = {{ .max_storage_in_gibibytes }}
			block_deployments_on_limit = {{ .block_deployments_on_limit }}
			email_notification = {{ .email_notification }}

			member {
				name  = "{{ .username1 }}"
				roles = ["developer","project admin"]
			}

			member {
				name  = "{{ .username2 }}"
				roles = ["developer"]
			}

			group {
				name  = "{{ .group1 }}"
				roles = ["qa"]
			}

			group {
				name  = "{{ .group2 }}"
				roles = ["release manager"]
			}

			role {
				name         = "qa"
				description  = "QA role"
				type         = "CUSTOM"
				environments = ["DEV"]
				actions      = ["READ_REPOSITORY","READ_RELEASE_BUNDLE", "READ_BUILD", "READ_SOURCES_PIPELINE", "READ_INTEGRATIONS_PIPELINE", "READ_POOLS_PIPELINE", "TRIGGER_PIPELINE"]
			}

			role {
				name         = "devop"
				description  = "DevOp role"
				type         = "CUSTOM"
				environments = ["DEV", "PROD"]
				actions      = ["READ_REPOSITORY", "ANNOTATE_REPOSITORY", "DEPLOY_CACHE_REPOSITORY", "DELETE_OVERWRITE_REPOSITORY", "TRIGGER_PIPELINE", "READ_INTEGRATIONS_PIPELINE", "READ_POOLS_PIPELINE", "MANAGE_INTEGRATIONS_PIPELINE", "MANAGE_SOURCES_PIPELINE", "MANAGE_POOLS_PIPELINE", "READ_BUILD", "ANNOTATE_BUILD", "DEPLOY_BUILD", "DELETE_BUILD",]
			}

			repos = ["{{ .repo1 }}", "{{ .repo2 }}"]
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTestUser(t, username1, email1)
			createTestUser(t, username2, email2)
			createTestGroup(t, group1)
			createTestGroup(t, group2)
			createTestRepo(t, repo1)
			createTestRepo(t, repo2)
		},
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			deleteTestUser(t, username1)
			deleteTestUser(t, username2)
			deleteTestGroup(t, group1)
			deleteTestGroup(t, group2)
			deleteTestRepo(t, repo1)
			deleteTestRepo(t, repo2)
			resp, err := verifyProject(id, request)

			return resp, err
		}),
		ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: project,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "max_storage_in_gibibytes", fmt.Sprintf("%d", params["max_storage_in_gibibytes"])),
					resource.TestCheckResourceAttr(resourceName, "block_deployments_on_limit", fmt.Sprintf("%t", params["block_deployments_on_limit"])),
					resource.TestCheckResourceAttr(resourceName, "email_notification", fmt.Sprintf("%t", params["email_notification"])),
					resource.TestCheckResourceAttr(resourceName, "admin_privileges.0.manage_members", fmt.Sprintf("%t", params["manage_members"])),
					resource.TestCheckResourceAttr(resourceName, "admin_privileges.0.manage_resources", fmt.Sprintf("%t", params["manage_resources"])),
					resource.TestCheckResourceAttr(resourceName, "admin_privileges.0.index_resources", fmt.Sprintf("%t", params["index_resources"])),
					resource.TestCheckResourceAttr(resourceName, "member.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "member.0.name", username1),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.0", "developer"),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.1", "project admin"),
					resource.TestCheckResourceAttr(resourceName, "member.1.name", username2),
					resource.TestCheckResourceAttr(resourceName, "member.1.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "member.1.roles.0", "developer"),
					resource.TestCheckResourceAttr(resourceName, "group.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "group.0.name", group1),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.0", "qa"),
					resource.TestCheckResourceAttr(resourceName, "group.1.name", group2),
					resource.TestCheckResourceAttr(resourceName, "group.1.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "group.1.roles.0", "release manager"),
					resource.TestCheckResourceAttr(resourceName, "repos.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "repos.*", repo1),
					resource.TestCheckTypeSetElemAttr(resourceName, "repos.*", repo2),
				),
			},
		},
	})
}
