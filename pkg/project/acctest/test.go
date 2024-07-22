package acctest

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	project "github.com/jfrog/terraform-provider-project/pkg/project"
	"github.com/jfrog/terraform-provider-shared/client"
	"github.com/jfrog/terraform-provider-shared/testutil"
)

// testAccProviderConfigure ensures Provider is only configured once
//
// The PreCheck(t) function is invoked for every test and this prevents
// extraneous reconfiguration to the same values each time. However, this does
// not prevent reconfiguration that may happen should the address of
// Provider be errantly reused in ProviderFactories.
var testAccProviderConfigure sync.Once

var Provider provider.Provider

var ProtoV6ProviderFactories map[string]func() (tfprotov6.ProviderServer, error)

func init() {
	Provider = project.Framework()()

	ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"project": providerserver.NewProtocol6WithError(Provider),
	}
}

// PreCheck This function should be present in every acceptance test.
func PreCheck(t *testing.T) {
	// Since we are outside the scope of the Terraform configuration we must
	// call Configure() to properly initialize the provider configuration.
	testAccProviderConfigure.Do(func() {
		restyClient := GetTestResty(t)

		artifactoryUrl := GetProjectUrl(t)
		// Set custom base URL so repos that relies on it will work
		// https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateCustomURLBase
		_, err := restyClient.R().
			SetBody(artifactoryUrl).
			SetHeader("Content-Type", "text/plain").
			Put("/artifactory/api/system/configuration/baseUrl")
		if err != nil {
			t.Fatalf("failed to set custom base URL: %v", err)
		}
	})
}

func GetProjectUrl(t *testing.T) string {
	return testutil.GetEnvVarWithFallback(t, "JFROG_URL", "PROJECT_URL")
}

type CheckFun func(id string, request *resty.Request) (*resty.Response, error)

func VerifyDeleted(id string, check CheckFun) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[id]
		if !ok {
			return fmt.Errorf("error: Resource id [%s] not found", id)
		}

		client := Provider.(*project.ProjectProvider).Meta.Client
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

func GetTestResty(t *testing.T) *resty.Client {
	var ok bool
	projectUrl := GetProjectUrl(t)
	restyClient, err := client.Build(projectUrl, "")
	if err != nil {
		t.Fatal(err)
	}

	var accessToken string
	if accessToken, ok = os.LookupEnv("PROJECT_ACCESS_TOKEN"); !ok {
		if accessToken, ok = os.LookupEnv("JFROG_ACCESS_TOKEN"); !ok {
			t.Fatal("PROJECT_ACCESS_TOKEN or JFROG_ACCESS_TOKEN must be set for acceptance tests")
		}
	}
	restyClient, err = client.AddAuth(restyClient, "", accessToken)
	if err != nil {
		t.Fatal(err)
	}

	return restyClient
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
