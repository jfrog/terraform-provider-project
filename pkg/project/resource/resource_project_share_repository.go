package project

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
)

const shareWithTargetProject = "access/api/v1/projects/_/share/repositories/{repo_key}/{target_project_key}"

func NewProjectShareRepositoryResource() resource.Resource {
	return &ProjectShareRepositoryResource{
		TypeName: "project_share_repository",
	}
}

type ProjectShareRepositoryResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ProjectShareRepositoryResourceModel struct {
	RepoKey          types.String `tfsdk:"repo_key"`
	TargetProjectKey types.String `tfsdk:"target_project_key"`
}

func (r *ProjectShareRepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ProjectShareRepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"target_project_key": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validatorfw_string.ProjectKey(),
				},
				Description: "The project key to which the repository should be shared with.",
			},
		},
		Description: "Share a local or remote repository with a list of projects. Project Members of the target project are granted actions to the shared repository according to their Roles and Role actions assigned in the target Project. Requires a user assigned with the 'Administer the Platform' role.\n\n" +
			"->Only available for Artifactory 7.90.1 or later.",
	}
}

func (r *ProjectShareRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectShareRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectShareRepositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectError ProjectErrorsResponse

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"repo_key":           plan.RepoKey.ValueString(),
			"target_project_key": plan.TargetProjectKey.ValueString(),
		}).
		SetError(&projectError).
		Put(shareWithTargetProject)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, projectError.String())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectShareRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectShareRepositoryResourceModel
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
		Get(ProjectRepositoryStatusEndpoint)

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

	if !slices.Contains(status.SharedWithProjects, state.TargetProjectKey.ValueString()) {
		state.TargetProjectKey = types.StringNull()
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectShareRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectShareRepositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectError ProjectErrorsResponse

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"repo_key":           plan.RepoKey.ValueString(),
			"target_project_key": plan.TargetProjectKey.ValueString(),
		}).
		SetError(&projectError).
		Put(shareWithTargetProject)

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, projectError.String())
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectShareRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectShareRepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectError ProjectErrorsResponse

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"repo_key":           state.RepoKey.ValueString(),
			"target_project_key": state.TargetProjectKey.ValueString(),
		}).
		SetError(&projectError).
		Delete(shareWithTargetProject)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, projectError.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

// ImportState imports the resource into the Terraform state.
func (r *ProjectShareRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected repo_key:target_project_key",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repo_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_project_key"), parts[1])...)
}
