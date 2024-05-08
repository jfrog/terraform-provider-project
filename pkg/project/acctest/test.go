package acctest

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	terraform2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-project/pkg/project/provider"
	"github.com/jfrog/terraform-provider-shared/client"
	"github.com/jfrog/terraform-provider-shared/testutil"
	"github.com/jfrog/terraform-provider-shared/util"
)

// Provider PreCheck(t) must be called before using this provider instance.
var Provider *schema.Provider
var ProviderFactories map[string]func() (*schema.Provider, error)

// testAccProviderConfigure ensures Provider is only configured once
//
// The PreCheck(t) function is invoked for every test and this prevents
// extraneous reconfiguration to the same values each time. However, this does
// not prevent reconfiguration that may happen should the address of
// Provider be errantly reused in ProviderFactories.
var testAccProviderConfigure sync.Once

// ProtoV6MuxProviderFactories is used to instantiate both SDK v2 and Framework providers
// during acceptance tests. Use it only if you need to combine resources from SDK v2 and the Framework in the same test.
var ProtoV6MuxProviderFactories map[string]func() (tfprotov6.ProviderServer, error)

var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"project": providerserver.NewProtocol6WithError(provider.Framework()()),
}

func init() {
	Provider = provider.SdkV2()

	ProviderFactories = map[string]func() (*schema.Provider, error){
		"project": func() (*schema.Provider, error) { return Provider, nil },
	}

	ProtoV6MuxProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"project": func() (tfprotov6.ProviderServer, error) {
			ctx := context.Background()

			upgradedSdkServer, err := tf5to6server.UpgradeServer(
				ctx,
				provider.SdkV2().GRPCProvider, // terraform-plugin-sdk provider
			)
			if err != nil {
				return nil, err
			}

			providers := []func() tfprotov6.ProviderServer{
				providerserver.NewProtocol6(provider.Framework()()), // terraform-plugin-framework provider
				func() tfprotov6.ProviderServer {
					return upgradedSdkServer
				},
			}

			muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)

			if err != nil {
				return nil, err
			}

			return muxServer.ProviderServer(), nil
		},
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

		configErr := Provider.Configure(context.Background(), (*terraform2.ResourceConfig)(terraform2.NewResourceConfigRaw(nil)))
		if configErr != nil && configErr.HasError() {
			t.Fatalf("failed to configure provider %v", configErr)
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

		client := Provider.Meta().(util.ProviderMetadata).Client
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
