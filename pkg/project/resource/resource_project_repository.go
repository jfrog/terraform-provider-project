package project

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
)

const repositoryEndpoint = "/artifactory/api/repositories/{key}"

func NewProjectRepositoryResource() resource.Resource {
	return &ProjectRepositoryResource{}
}

type ProjectRepositoryResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ProjectRepositoryResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Key        types.String `tfsdk:"key"`
	ProjectKey types.String `tfsdk:"project_key"`
}

type ProjectRepositoryAPIModel struct {
	Key        string `json:"key"`
	ProjectKey string `json:"projectKey"`
}

func (r *ProjectRepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
	r.TypeName = resp.TypeName
}

func (r *ProjectRepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"key": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validatorfw_string.RepoKey(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The key of the repository.",
			},
			"project_key": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validatorfw_string.ProjectKey(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The key of the project to which the repository should be assigned to.",
			},
		},
		Description: "Assign a repository to a project. Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.",
	}
}

func (r *ProjectRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ProjectRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectRepositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	repoKey := plan.Key.ValueString()

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"repoKey":    repoKey,
		}).
		SetError(&projectError).
		Put("/access/api/v1/projects/_/attach/repositories/{repoKey}/{projectKey}?force=true")
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}
	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, projectError.String())
		return
	}

	var retryFunc = func() error {
		var repo ProjectRepositoryAPIModel
		resp, err := r.ProviderData.Client.R().
			SetResult(&repo).
			SetPathParam("key", repoKey).
			Get(repositoryEndpoint)

		if err != nil {
			return fmt.Errorf("error getting repository: %s", err)
		}
		if resp.IsError() {
			return fmt.Errorf("error getting repository: %s", resp.String())
		}

		if repo.ProjectKey == "" {
			return fmt.Errorf("expected repository to be assigned to project but currently not")
		}

		return nil
	}

	bf := backoff.WithContext(
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(20*time.Minute)),
		ctx,
	)
	retryError := backoff.Retry(retryFunc, bf)
	if retryError != nil {
		utilfw.UnableToCreateResourceError(resp, retryError.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s-%s", projectKey, repoKey))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectRepositoryResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := state.ProjectKey.ValueString()
	repoKey := state.Key.ValueString()

	var repo ProjectRepositoryAPIModel
	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetResult(&repo).
		SetPathParam("key", repoKey).
		Get(repositoryEndpoint)
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusBadRequest || response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, projectError.String())
		return
	}
	if repo.ProjectKey == "" {
		tflog.Warn(ctx, "no project_key for repo", map[string]any{"repoKey": repoKey})
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s-%s", projectKey, repoKey))
	state.ProjectKey = types.StringValue(projectKey)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddWarning(
		"Update not supported",
		"Repository assignment to project cannnot be updated.",
	)
}

func (r *ProjectRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectRepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParam("repoKey", state.Key.ValueString()).
		SetError(&projectError).
		Delete("/access/api/v1/projects/_/attach/repositories/{repoKey}")
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}
	if response.IsError() && response.StatusCode() != http.StatusNotFound {
		utilfw.UnableToDeleteResourceError(resp, projectError.String())
		return
	}

	// If the logic reaches here, it implicitly succeeded and will remove
	// the resource from state if there are no other errors.
}

// ImportState imports the resource into the Terraform state.
func (r *ProjectRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected project_key:repository_key",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[1])...)
}
