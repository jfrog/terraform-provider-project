package project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	project "github.com/jfrog/terraform-provider-project/pkg/project/resource"
	"github.com/jfrog/terraform-provider-shared/client"
	"github.com/jfrog/terraform-provider-shared/util"
	validatorfw_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
)

var Version = "1.9.1"

// needs to be exported so make file can update this
var productId = "terraform-provider-project/" + Version

// Ensure the implementation satisfies the provider.Provider interface.
var _ provider.Provider = &ProjectProvider{}

type ProjectProvider struct {
	Meta util.ProviderMetadata
}

// ProjectProviderModel describes the provider data model.
type ProjectProviderModel struct {
	Url                  types.String `tfsdk:"url"`
	AccessToken          types.String `tfsdk:"access_token"`
	OIDCProviderName     types.String `tfsdk:"oidc_provider_name"`
	TFCCredentialTagName types.String `tfsdk:"tfc_credential_tag_name"`
	CheckLicense         types.Bool   `tfsdk:"check_license"`
}

// Metadata satisfies the provider.Provider interface for ProjectProvider
func (p *ProjectProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "project"
	resp.Version = Version
}

// Schema satisfies the provider.Provider interface for ProjectProvider.
func (p *ProjectProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					validatorfw_string.IsURLHttpOrHttps(),
				},
				Description: "URL of Artifactory. This can also be sourced from the `PROJECT_URL` or `JFROG_URL` environment variable. Default to 'http://localhost:8081' if not set.",
			},
			"access_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "This is a Bearer token that can be given to you by your admin under `Identity and Access`. This can also be sourced from the `PROJECT_ACCESS_TOKEN` or `JFROG_ACCESS_TOKEN` environment variable. Defauult to empty string if not set.",
			},
			"oidc_provider_name": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "OIDC provider name. See [Configure an OIDC Integration](https://jfrog.com/help/r/jfrog-platform-administration-documentation/configure-an-oidc-integration) for more details.",
			},
			"tfc_credential_tag_name": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "Terraform Cloud Workload Identity Token tag name. Use for generating multiple TFC workload identity tokens. When set, the provider will attempt to use env var with this tag name as suffix. **Note:** this is case sensitive, so if set to `JFROG`, then env var `TFC_WORKLOAD_IDENTITY_TOKEN_JFROG` is used instead of `TFC_WORKLOAD_IDENTITY_TOKEN`. See [Generating Multiple Tokens](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials/manual-generation#generating-multiple-tokens) on HCP Terraform for more details.",
			},
			"check_license": schema.BoolAttribute{
				Optional:           true,
				Description:        "Toggle for pre-flight checking of Artifactory Enterprise license. Default to `true`.",
				DeprecationMessage: "Remove this attribute from your provider configuration as it is no longer used and the attribute will be removed in the next major version of the provider.",
			},
		},
	}
}

func (p *ProjectProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Check environment variables, first available OS variable will be assigned to the var
	url := util.CheckEnvVars([]string{"JFROG_URL", "PROJECT_URL"}, "")
	accessToken := util.CheckEnvVars([]string{"JFROG_ACCESS_TOKEN", "PROJECT_ACCESS_TOKEN"}, "")

	var config ProjectProviderModel

	// Read configuration data into model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Url.ValueString() != "" {
		url = config.Url.ValueString()
	}

	if url == "" {
		resp.Diagnostics.AddError(
			"Missing URL Configuration",
			"While configuring the provider, the url was not found in "+
				"the JFROG_URL/ARTIFACTORY_URL environment variable or provider "+
				"configuration block url attribute.",
		)
		return
	}

	restyClient, err := client.Build(url, productId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Resty client",
			err.Error(),
		)
		return
	}

	oidcProviderName := config.OIDCProviderName.ValueString()
	if oidcProviderName != "" {
		oidcAccessToken, err := util.OIDCTokenExchange(ctx, restyClient, oidcProviderName, config.TFCCredentialTagName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed OIDC ID token exchange",
				err.Error(),
			)
			return
		}

		// use token from OIDC provider, which should take precedence over
		// environment variable data, if found.
		if oidcAccessToken != "" {
			accessToken = oidcAccessToken
		}
	}

	// Check configuration data, which should take precedence over
	// environment variable data or OIDC access token, if found.
	if config.AccessToken.ValueString() != "" {
		accessToken = config.AccessToken.ValueString()
	}

	if accessToken == "" {
		resp.Diagnostics.AddError(
			"Missing JFrog Access Token",
			"While configuring the provider, the Access Token was not found in "+
				"the JFROG_ACCESS_TOKEN/PROJECT_ACCESS_TOKEN environment variable or provider "+
				"configuration block access_token attribute.",
		)
		return
	}

	restyClient, err = client.AddAuth(restyClient, "", accessToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error adding Auth to Resty client",
			err.Error(),
		)
	}

	version, err := util.GetArtifactoryVersion(restyClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Artifactory version",
			fmt.Sprintf("The provider functionality might be affected by the absence of Artifactory version in the context. %v", err),
		)
		return
	}

	featureUsage := fmt.Sprintf("Terraform/%s", req.TerraformVersion)
	go util.SendUsage(ctx, restyClient.R(), productId, featureUsage)

	meta := util.ProviderMetadata{
		Client:             restyClient,
		ProductId:          productId,
		ArtifactoryVersion: version,
	}

	p.Meta = meta

	resp.DataSourceData = meta
	resp.ResourceData = meta
}

// Resources satisfies the provider.Provider interface for ProjectProvider.
func (p *ProjectProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		project.NewProjectResource,
		project.NewProjectEnvironmentResource,
		project.NewProjectGroupResource,
		project.NewProjectRepositoryResource,
		project.NewProjectRoleResource,
		project.NewProjectShareRepositoryResource,
		project.NewProjectShareRepositoryWithAllResource,
		project.NewProjectUserResource,
	}
}

// DataSources satisfies the provider.Provider interface for ProjectProvider.
func (p *ProjectProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func NewProvider() func() provider.Provider {
	return func() provider.Provider {
		return &ProjectProvider{}
	}
}
