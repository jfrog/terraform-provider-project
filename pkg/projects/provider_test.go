package projects

import (
	"context"
	"github.com/go-resty/resty/v2"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAccProviders = func() map[string]func() (*schema.Provider, error) {
	provider := Provider()
	return map[string]func() (*schema.Provider, error){
		"projects": func() (*schema.Provider, error) {
			return provider, nil
		},
	}
}()

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}
func getTestResty(t *testing.T) *resty.Client {
	if v := os.Getenv("PROJECTS_URL"); v == "" {
		t.Fatal("PROJECTS_URL must be set for acceptance tests")
	}
	restyClient, err := buildResty(os.Getenv("PROJECTS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	accessToken := os.Getenv("PROJECTS_ACCESS_TOKEN")
	restyClient, err = addAuthToResty(restyClient, accessToken)
	if err != nil {
		t.Fatal(err)
	}
	return restyClient
}

func testAccPreCheck(t *testing.T) {
	restyClient := getTestResty(t)
	// TODO check the payload and make sure it's the right license type
	_, err := restyClient.R().Get("/artifactory/api/system/licenses/")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	provider, _ := testAccProviders["projects"]()
	oldErr := provider.Configure(ctx, terraform.NewResourceConfigRaw(nil))
	if oldErr != nil {
		t.Fatal(oldErr)
	}
}
