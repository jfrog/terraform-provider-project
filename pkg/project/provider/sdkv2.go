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
	"github.com/jfrog/terraform-provider-shared/validator"
)

// Provider Projects provider that supports configuration via a token
// Supported resources are repos, users, groups, replications, and permissions
func SdkV2() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				Description:  "URL of Artifactory. This can also be sourced from the `PROJECT_URL` or `JFROG_URL` environment variable. Default to 'http://localhost:8081' if not set.",
			},
			"access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "This is a Bearer token that can be given to you by your admin under `Identity and Access`. This can also be sourced from the `PROJECT_ACCESS_TOKEN` or `JFROG_ACCESS_TOKEN` environment variable. Defauult to empty string if not set.",
			},
			"oidc_provider_name": {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validator.StringIsNotEmpty,
				Description:      "OIDC provider name. See [Configure an OIDC Integration](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-an-oidc-integration) for more details.",
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
				"project_role":       resource.ProjectRoleResource(),
				"project_repository": resource.ProjectRepositoryResource(),
			},
		),
	}

	p.ConfigureContextFunc = func(ctx context.Context, data *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, data, p.TerraformVersion)
	}

	return p
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	url := util.CheckEnvVars([]string{"JFROG_URL", "PROJECT_URL"}, "")
	accessToken := util.CheckEnvVars([]string{"JFROG_ACCESS_TOKEN", "PROJECT_ACCESS_TOKEN"}, "")

	if v, ok := d.GetOk("url"); ok {
		url = v.(string)
	}
	if url == "" {
		return nil, diag.Errorf("missing URL Configuration")
	}

	restyClient, err := client.Build(url, productId)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	if v, ok := d.GetOk("oidc_provider_name"); ok {
		oidcAccessToken, err := util.OIDCTokenExchange(ctx, restyClient, v.(string))
		if err != nil {
			return nil, diag.FromErr(err)
		}

		if oidcAccessToken != "" {
			accessToken = oidcAccessToken
		}
	}

	if v, ok := d.GetOk("access_token"); ok && v != "" {
		accessToken = v.(string)
	}

	if accessToken == "" {
		return nil, diag.Errorf("Missing JFrog Access Token\n" +
			"While configuring the provider, the Access Token was not found in " +
			"the JFROG_ACCESS_TOKEN/PROJECT_ACCESS_TOKEN environment variable or provider " +
			"configuration block access_token attribute.")
	}

	restyClient, err = client.AddAuth(restyClient, "", accessToken)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	checkLicense := d.Get("check_license").(bool)
	if checkLicense {
		licenseErr := util.CheckArtifactoryLicense(restyClient, "Enterprise", "Commercial", "Edge")
		if licenseErr != nil {
			return nil, diag.FromErr(licenseErr)
		}
	}

	version, err := util.GetArtifactoryVersion(restyClient)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	featureUsage := fmt.Sprintf("Terraform/%s", terraformVersion)
	go util.SendUsage(ctx, restyClient.R(), productId, featureUsage)

	return util.ProviderMetadata{
		Client:             restyClient,
		ArtifactoryVersion: version,
	}, nil
}
