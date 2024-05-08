package provider_test

import (
	"testing"

	provider "github.com/jfrog/terraform-provider-project/pkg/project/provider"
)

func TestProvider_validate(t *testing.T) {
	if err := provider.SdkV2().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = provider.SdkV2()
}
