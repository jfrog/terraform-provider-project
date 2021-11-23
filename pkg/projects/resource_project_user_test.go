package projects

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

type ArtifactoryUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

func createTestUser(t *testing.T, projectKey string, username string, email string) {
	restyClient := getTestResty(t)

	user := ArtifactoryUser{
		Email:    email,
		Password: "Password1",
		Admin:    false,
	}

	_, err := restyClient.R().SetBody(user).Put(fmt.Sprintf("/artifactory/api/security/users/%s", username))
	if err != nil {
		t.Fatal(err)
	}
}

func deleteTestUser(t *testing.T, projectKey string, username string) {
	restyClient := getTestResty(t)

	_, err := restyClient.R().Delete(fmt.Sprintf("/artifactory/api/security/users/%s", username))
	if err != nil {
		t.Fatal(err)
	}
}

func TestAccProjectUserAdd(t *testing.T) {
	name := fmt.Sprintf("tftestprojects%s", randSeq(10))
	resourceName := fmt.Sprintf("project_project.%s", name)
	projectKey := strings.ToLower(randSeq(6))

	username := "user1"
	email := "user1@tempurl.org"
	role := "developer"

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"username":    username,
		"role":        role,
	}
	updateConfig := executeTemplate("TestAccProjectUser", `
		resource "project_project" "{{ .name }}" {
            key = "{{ .project_key }}"
            display_name = "{{ .name }}"
            description = "test description"
            admin_privileges {
                manage_members = true
                manage_resources = true
                index_resources = true
            }

			user {
				name = "{{ .username }}"
				roles = ["{{ .role }}"]
			}
        }
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTestUser(t, projectKey, username, email)
		},
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			deleteTestUser(t, projectKey, username)
			resp, err := verifyProject(id, request)

			return resp, err
		}),
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "user.0.name", username),
					resource.TestCheckResourceAttr(resourceName, "user.0.roles.0", role),
				),
			},
		},
	})
}

func TestAccProjectUserUpdate(t *testing.T) {
	name := fmt.Sprintf("tftestprojects%s", randSeq(10))
	resourceName := fmt.Sprintf("project_project.%s", name)
	projectKey := strings.ToLower(randSeq(6))

	username := "user1"
	email := "user1@tempurl.org"
	role := "developer"
	newRole := "contributor"

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"username":    username,
		"role":        role,
		"newRole":        newRole,
	}
	updateConfig := executeTemplate("TestAccProjectUser", `
		resource "project_project" "{{ .name }}" {
            key = "{{ .project_key }}"
            display_name = "{{ .name }}"
            description = "test description"
            admin_privileges {
                manage_members = true
                manage_resources = true
                index_resources = true
            }

			user {
				name = "{{ .username }}"
				roles = ["{{ .role }}"]
			}
        }
	`, params)

	projectConfig := executeTemplate("TestAccProjectUser", `
		resource "project_project" "{{ .name }}" {
            key = "{{ .project_key }}"
            display_name = "{{ .name }}"
            description = "test description"
            admin_privileges {
                manage_members = true
                manage_resources = true
                index_resources = true
            }

			user {
				name = "{{ .username }}"
				roles = ["{{ .newRole }}"]
			}
        }
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTestUser(t, projectKey, username, email)
		},
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			deleteTestUser(t, projectKey, username)
			resp, err := verifyProject(id, request)

			return resp, err
		}),
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "user.0.name", username),
					resource.TestCheckResourceAttr(resourceName, "user.0.roles.0", role),
				),
			},
			{
				Config: projectConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "user.0.name", username),
					resource.TestCheckResourceAttr(resourceName, "user.0.roles.0", newRole),
				),
			},
		},
	})
}

func TestAccProjectUserDelete(t *testing.T) {
	name := fmt.Sprintf("tftestprojects%s", randSeq(10))
	resourceName := fmt.Sprintf("project_project.%s", name)
	projectKey := strings.ToLower(randSeq(6))

	username := "user1"
	email := "user1@tempurl.org"
	role := "developer"

	params := map[string]interface{}{
		"name":        name,
		"project_key": projectKey,
		"username":    username,
		"role":        role,
	}
	updateConfig := executeTemplate("TestAccProjectUser", `
		resource "project_project" "{{ .name }}" {
            key = "{{ .project_key }}"
            display_name = "{{ .name }}"
            description = "test description"
            admin_privileges {
                manage_members = true
                manage_resources = true
                index_resources = true
            }

			user {
				name = "{{ .username }}"
				roles = ["{{ .role }}"]
			}
        }
	`, params)

	projectConfig := executeTemplate("TestAccProjectUser", `
		resource "project_project" "{{ .name }}" {
            key = "{{ .project_key }}"
            display_name = "{{ .name }}"
            description = "test description"
            admin_privileges {
                manage_members = true
                manage_resources = true
                index_resources = true
            }
        }
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTestUser(t, projectKey, username, email)
		},
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			deleteTestUser(t, projectKey, username)
			resp, err := verifyProject(id, request)

			return resp, err
		}),
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "user.0.name", username),
					resource.TestCheckResourceAttr(resourceName, "user.0.roles.0", role),
				),
			},
			{
				Config: projectConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckNoResourceAttr(resourceName, "user"),
				),
			},
		},
	})
}
