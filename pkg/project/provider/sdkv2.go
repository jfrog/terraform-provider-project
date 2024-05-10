package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	resource "github.com/jfrog/terraform-provider-project/pkg/project/resource"
	"github.com/jfrog/terraform-provider-shared/client"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/jfrog/terraform-provider-shared/util/sdk"
)

// Provider Projects provider that supports configuration via a token
// Supported resources are repos, users, groups, replications, and permissions
func SdkV2() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"PROJECT_URL", "JFROG_URL"}, "http://localhost:8081"),
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				Description:  "URL of Artifactory. This can also be sourced from the `PROJECT_URL` or `JFROG_URL` environment variable. Default to 'http://localhost:8081' if not set.",
			},
			"access_token": {
				Type:             schema.TypeString,
				Required:         true,
				Sensitive:        true,
				DefaultFunc:      schema.MultiEnvDefaultFunc([]string{"PROJECT_ACCESS_TOKEN", "JFROG_ACCESS_TOKEN"}, ""),
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "This is a Bearer token that can be given to you by your admin under `Identity and Access`. This can also be sourced from the `PROJECT_ACCESS_TOKEN` or `JFROG_ACCESS_TOKEN` environment variable. Defauult to empty string if not set.",
			},
			"check_license": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Toggle for pre-flight checking of Artifactory Enterprise license. Default to `true`.",
			},
		},

		ResourcesMap: sdk.AddTelemetry(
			productId,
			map[string]*schema.Resource{
				"project_environment": resource.ProjectEnvironmentResource(),
				"project_role":        resource.ProjectRoleResource(),
				"project_user":        resource.ProjectUserResource(),
				"project_group":       resource.ProjectGroupResource(),
				"project_repository":  resource.ProjectRepositoryResource(),
			},
		),
	}

	p.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, data, p.TerraformVersion)
	}

	return p
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	URL, ok := d.GetOk("url")
	if URL == nil || URL == "" || !ok {
		return nil, diag.Errorf("you must supply a URL")
	}

	restyBase, err := client.Build(URL.(string), productId)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	accessToken := d.Get("access_token").(string)
	restyBase, err = client.AddAuth(restyBase, "", accessToken)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	checkLicense := d.Get("check_license").(bool)
	if checkLicense {
		licenseErr := util.CheckArtifactoryLicense(restyBase, "Enterprise", "Commercial", "Edge")
		if licenseErr != nil {
			return nil, diag.FromErr(licenseErr)
		}
	}

	version, err := util.GetArtifactoryVersion(restyBase)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	featureUsage := fmt.Sprintf("Terraform/%s", terraformVersion)
	util.SendUsage(ctx, restyBase.R(), productId, featureUsage)

	return util.ProviderMetadata{
		Client:             restyBase,
		ArtifactoryVersion: version,
	}, nil
}
