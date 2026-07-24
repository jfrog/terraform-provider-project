package project_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/jfrog/terraform-provider-project/pkg/project"
)

// TestProvider_resourceNamespacing verifies Phase 1 of issue #210: every
// resource is exposed under the new `jfrog_` namespaced name, and the legacy
// un-namespaced name is still registered as a deprecated alias pointing at the
// same implementation.
func TestProvider_resourceNamespacing(t *testing.T) {
	ctx := context.Background()

	p, ok := project.NewProvider()().(interface {
		Resources(context.Context) []func() resource.Resource
	})
	if !ok {
		t.Fatal("provider does not expose Resources")
	}

	// Expected legacy name -> namespaced name pairs.
	expected := map[string]string{
		"project":                           "jfrog_project",
		"project_environment":               "jfrog_project_environment",
		"project_group":                     "jfrog_project_group",
		"project_repository":                "jfrog_project_repository",
		"project_role":                      "jfrog_project_role",
		"project_share_repository":          "jfrog_project_share_repository",
		"project_share_repository_with_all": "jfrog_project_share_repository_with_all",
		"project_user":                      "jfrog_project_user",
	}

	type meta struct {
		deprecated bool
	}
	seen := map[string]meta{}

	for _, rf := range p.Resources(ctx) {
		res := rf()

		var metaResp resource.MetadataResponse
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "project"}, &metaResp)

		var schemaResp resource.SchemaResponse
		res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

		if _, dup := seen[metaResp.TypeName]; dup {
			t.Fatalf("resource type name %q registered more than once", metaResp.TypeName)
		}
		seen[metaResp.TypeName] = meta{deprecated: schemaResp.Schema.GetDeprecationMessage() != ""}
	}

	for legacy, namespaced := range expected {
		ns, ok := seen[namespaced]
		if !ok {
			t.Errorf("expected namespaced resource %q to be registered", namespaced)
		} else if ns.deprecated {
			t.Errorf("namespaced resource %q must NOT be deprecated", namespaced)
		}

		lg, ok := seen[legacy]
		if !ok {
			t.Errorf("expected legacy resource %q to remain registered (deprecated alias)", legacy)
		} else if !lg.deprecated {
			t.Errorf("legacy resource %q must be marked deprecated", legacy)
		}
	}

	if len(seen) != len(expected)*2 {
		t.Errorf("expected %d registered resources, got %d", len(expected)*2, len(seen))
	}

	for name := range seen {
		if name == "" {
			t.Error("found a resource with an empty type name")
		}
		if !strings.HasPrefix(name, "project") && !strings.HasPrefix(name, "jfrog_project") {
			t.Errorf("unexpected resource type name %q", name)
		}
	}
}
