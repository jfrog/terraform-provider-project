package project

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	terraform2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-shared/util"
)

// Provider PreCheck(t) must be called before using this provider instance.
var TestProvider *schema.Provider

var ProviderFactories map[string]func() (*schema.Provider, error)

// testAccProviderConfigure ensures Provider is only configured once
//
// The PreCheck(t) function is invoked for every test and this prevents
// extraneous reconfiguration to the same values each time. However, this does
// not prevent reconfiguration that may happen should the address of
// Provider be errantly reused in ProviderFactories.
var testAccProviderConfigure sync.Once

func init() {
	TestProvider = Provider()
	ProviderFactories = map[string]func() (*schema.Provider, error){
		"project": func() (*schema.Provider, error) { return TestProvider, nil },
	}
}

func testAccPreCheck(t *testing.T) {
	testAccProviderConfigure.Do(func() {
		err := TestProvider.Configure(context.Background(), terraform2.NewResourceConfigRaw(nil))
		if err != nil && err.HasError() {
			t.Fatal(err)
		}
	})
}

type CheckFun func(id string, request *resty.Request) (*resty.Response, error)

func verifyDeleted(id string, check CheckFun) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("error: Resource id [%s] not found", id)
		}

		client := TestProvider.Meta().(util.ProviderMetadata).Client
		resp, err := check(rs.Primary.ID, client.R())
		if err != nil {
			return err
		}

		if resp != nil {
			switch resp.StatusCode() {
			case http.StatusNotFound, http.StatusBadRequest:
				return nil
			}
		}

		return fmt.Errorf("error: %s still exists: %d", rs.Primary.ID, resp.StatusCode())
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
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
