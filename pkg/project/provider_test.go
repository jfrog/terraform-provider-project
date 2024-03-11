package project

import (
	"os"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jfrog/terraform-provider-shared/client"
)

func TestProvider_validate(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}

func getTestResty(t *testing.T) *resty.Client {
	var ok bool
	var projectUrl string
	if projectUrl, ok = os.LookupEnv("PROJECT_URL"); !ok {
		if projectUrl, ok = os.LookupEnv("JFROG_URL"); !ok {
			t.Fatal("PROJECT_URL or JFROG_URL must be set for acceptance tests")
		}
	}
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
