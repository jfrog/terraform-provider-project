package projects

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

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
	resp, err := restyClient.R().Get("/artifactory/api/system/licenses/")
	if err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf("%s", resp.Body())
	if !strings.Contains(body, "Enterprise") {
		t.Fatal(body, "\nArtifactory requires Enterprise license to work with Terraform!")
	}

	ctx := context.Background()
	provider, _ := testAccProviders["project"]()
	oldErr := provider.Configure(ctx, terraform.NewResourceConfigRaw(nil))
	if oldErr != nil {
		t.Fatal(oldErr)
	}
}
