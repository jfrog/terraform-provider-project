package project

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testAccProviders() map[string]func() (*schema.Provider, error) {
	return map[string]func() (*schema.Provider, error){
		"project": func() (*schema.Provider, error) {
			return Provider(), nil
		},
	}
}

type CheckFun func(id string, request *resty.Request) (*resty.Response, error)

func verifyDeleted(id string, check CheckFun) func(*terraform.State) error {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[id]

		if !ok {
			return fmt.Errorf("error: Resource id [%s] not found", id)
		}
		provider, _ := testAccProviders()["project"]()
		provider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
		client := provider.Meta().(*resty.Client)
		resp, err := check(rs.Primary.ID, client.R())
		if err != nil {
			if resp != nil {
				switch resp.StatusCode() {
				case http.StatusNotFound, http.StatusBadRequest:
					return nil
				}
			}
			return err
		}
		return fmt.Errorf("error: %s still exists", rs.Primary.ID)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func createTestUser(t *testing.T, name string, email string) {

	type ArtifactoryUser struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Admin    bool   `json:"admin"`
	}

	restyClient := getTestResty(t)

	user := ArtifactoryUser{
		Email:    email,
		Password: "Password1!",
		Admin:    false,
	}

	_, err := restyClient.R().SetBody(user).Put("/artifactory/api/security/users/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func deleteTestUser(t *testing.T, name string) {
	restyClient := getTestResty(t)

	_, err := restyClient.R().Delete("/artifactory/api/security/users/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func createTestGroup(t *testing.T, name string) {

	type ArtifactoryGroup struct {
		Name string `json:"name"`
	}

	restyClient := getTestResty(t)

	group := ArtifactoryGroup{
		Name: name,
	}

	_, err := restyClient.R().SetBody(group).Put("/artifactory/api/security/groups/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func deleteTestGroup(t *testing.T, name string) {
	restyClient := getTestResty(t)

	_, err := restyClient.R().Delete("/artifactory/api/security/groups/" + name)
	if err != nil {
		t.Fatal(err)
	}
}

func createTestRepo(t *testing.T, name string) {

	type ArtifactoryRepo struct {
		Name   string `json:"key"`
		RClass string `json:"rclass"`
	}

	restyClient := getTestResty(t)

	repo := ArtifactoryRepo{
		Name:   name,
		RClass: "local",
	}

	_, err := restyClient.
		R().
		AddRetryCondition(retryOnSpecificMsgBody("A timeout occurred")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is down")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is returning an unknown error")).
		SetBody(repo).
		Put("/artifactory/api/repositories/" + name)

	if err != nil {
		t.Fatal(err)
	}
}

func deleteTestRepo(t *testing.T, name string) {
	restyClient := getTestResty(t)
	_, err := restyClient.
		R().
		AddRetryCondition(retryOnSpecificMsgBody("A timeout occurred")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is down")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is returning an unknown error")).
		Delete("/artifactory/api/repositories/" + name)

	if err != nil {
		t.Fatal(err)
	}
}
