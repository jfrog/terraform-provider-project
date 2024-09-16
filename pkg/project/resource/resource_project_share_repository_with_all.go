package project

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
)

const shareWithAllProjectsEndpoint = "access/api/v1/projects/_/share/repositories/{repo_key}"

func NewProjectShareRepositoryWithAllResource() resource.Resource {
	return &ProjectShareRepositoryWithAllResource{
		TypeName: "project_share_repository_with_all",
	}
}

type ProjectShareRepositoryWithAllResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ProjectShareRepositoryWithAllResourceModel struct {
	RepoKey  types.String `tfsdk:"repo_key"`
	ReadOnly types.Bool   `tfsdk:"read_only"`
}

func (r *ProjectShareRepositoryWithAllResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ProjectShareRepositoryWithAllResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"repo_key": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validatorfw_string.RepoKey(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The key of the repository.",
			},
			"read_only": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
				Description: "Share repository with all Projects in Read-Only mode to avoid any changes or modifications of the shared content.\n\n" +
					"->Only available for Artifactory 7.94.0 or later.",
			},
		},
		Description: "Share a local or remote repository with all projects. Project Members of the target project are granted actions to the shared repository according to their Roles and Role actions assigned in the target Project. Requires a user assigned with the 'Administer the Platform' role.\n\n" +
			"->Only available for Artifactory 7.90.1 or later.",
	}
}

func (r *ProjectShareRepositoryWithAllResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	r.ProviderData = req.ProviderData.(util.ProviderMetadata)

	supported, err := util.CheckVersion(r.ProviderData.ArtifactoryVersion, "7.90.1")
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to check Artifactory version",
			err.Error(),
		)
		return
	}

	if !supported {
		resp.Diagnostics.AddError(
			"Unsupported Artifactory version",
			fmt.Sprintf("This resource is supported by Artifactory version 7.90.1 or later. Current version: %s", r.ProviderData.ArtifactoryVersion),
		)
		return
	}
}

func (r *ProjectShareRepositoryWithAllResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectShareRepositoryWithAllResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectError ProjectErrorsResponse

	response, err := r.ProviderData.Client.R().
		SetPathParam("repo_key", plan.RepoKey.ValueString()).
		SetQueryParam("readOnly", fmt.Sprintf("%t", plan.ReadOnly.ValueBool())).
		SetError(&projectError).
		Put(shareWithAllProjectsEndpoint)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, projectError.String())
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectShareRepositoryWithAllResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectShareRepositoryWithAllResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := state.RepoKey.ValueString()

	var status ProjectRepositoryStatusAPIModel
	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParam("repo_key", repoKey).
		SetResult(&status).
		SetError(&projectError).
		Get("access/api/v1/projects/_/repositories/{repo_key}")

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		resp.Diagnostics.AddWarning(
			"repo not found",
			repoKey,
		)
		resp.State.RemoveResource(ctx)
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, projectError.String())
		return
	}

	if !status.SharedWithAllProjects {
		resp.Diagnostics.AddWarning(
			"repo is not shared with all projects",
			repoKey,
		)
		resp.State.RemoveResource(ctx)
		return
	}

	state.ReadOnly = types.BoolValue(status.SharedReadOnly)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectShareRepositoryWithAllResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddWarning(
		"Update not supported",
		"Repository sharing with all projects cannnot be updated.",
	)
}

func (r *ProjectShareRepositoryWithAllResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectShareRepositoryWithAllResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectError ProjectErrorsResponse

	response, err := r.ProviderData.Client.R().
		SetPathParam("repo_key", state.RepoKey.ValueString()).
		SetQueryParam("readOnly", fmt.Sprintf("%t", state.ReadOnly.ValueBool())).
		SetError(&projectError).
		Delete(shareWithAllProjectsEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, projectError.String())
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

// ImportState imports the resource into the Terraform state.
func (r *ProjectShareRepositoryWithAllResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("repo_key"), req, resp)
}
