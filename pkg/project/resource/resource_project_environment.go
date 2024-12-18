package project

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/samber/lo"
)

const ProjectEnvironmentUrl = "/access/api/v1/projects/{projectKey}/environments"

func NewProjectEnvironmentResource() resource.Resource {
	return &ProjectEnvironmentResource{
		TypeName: "project_environment",
	}
}

type ProjectEnvironmentResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ProjectEnvironmentResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	ProjectKey types.String `tfsdk:"project_key"`
}

type ProjectEnvironmentAPIModel struct {
	Name string `json:"name"`
}

type ProjectEnvironmentUpdateAPIModel struct {
	NewName string `json:"new_name"`
}

func (r *ProjectEnvironmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ProjectEnvironmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]+$`), "Must start with a letter and contain letters, digits and `-` character."),
				},
				Description: "Environment name. Must start with a letter and can contain letters, digits and `-` character.",
			},
			"project_key": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validatorfw_string.ProjectKey(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Project key for this environment. This field supports only 2 - 32 lowercase alphanumeric and hyphen characters. Must begin with a letter.",
			},
		},
		Description: "Creates a new environment for the specified project.\n\n~>The combined length of `project_key` and `name` (separated by '-') cannot not exceeds 32 characters.",
	}
}

func (r *ProjectEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ProjectEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectEnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()

	environment := ProjectEnvironmentAPIModel{
		Name: fmt.Sprintf("%s-%s", projectKey, plan.Name.ValueString()),
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParam("projectKey", projectKey).
		SetBody(environment).
		SetError(&projectError).
		Post(ProjectEnvironmentUrl)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}
	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, projectError.String())
		return
	}

	plan.ID = types.StringValue(environment.Name)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectEnvironmentResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := state.ProjectKey.ValueString()

	var environments []ProjectEnvironmentAPIModel
	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParam("projectKey", projectKey).
		SetResult(&environments).
		SetError(&projectError).
		Get(ProjectEnvironmentUrl)
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}
	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, projectError.String())
		return
	}

	matchedEnv, ok := lo.Find(environments, func(env ProjectEnvironmentAPIModel) bool {
		return env.Name == fmt.Sprintf("%s-%s", projectKey, state.Name.ValueString())
	})
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	environmentName := strings.TrimPrefix(matchedEnv.Name, fmt.Sprintf("%s-", projectKey))
	state.ID = types.StringValue(matchedEnv.Name)
	state.Name = types.StringValue(environmentName)
	state.ProjectKey = types.StringValue(projectKey)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectEnvironmentResourceModel
	var state ProjectEnvironmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldName := state.Name.ValueString()
	newName := plan.Name.ValueString()
	projectKey := plan.ProjectKey.ValueString()

	environmentUpdate := ProjectEnvironmentUpdateAPIModel{
		NewName: fmt.Sprintf("%s-%s", projectKey, newName),
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey":      projectKey,
			"environmentName": fmt.Sprintf("%s-%s", projectKey, oldName),
		}).
		SetBody(environmentUpdate).
		SetError(&projectError).
		Post(ProjectEnvironmentUrl + "/{environmentName}/rename")
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}
	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, projectError.String())
		return
	}

	plan.ID = types.StringValue(environmentUpdate.NewName)
	plan.Name = types.StringValue(newName)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectEnvironmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := state.ProjectKey.ValueString()

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey":      projectKey,
			"environmentName": fmt.Sprintf("%s-%s", projectKey, state.Name.ValueString()),
		}).
		SetError(&projectError).
		Delete(ProjectEnvironmentUrl + "/{environmentName}")
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
func (r *ProjectEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected project_key:environment_name",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

func (r ProjectEnvironmentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ProjectEnvironmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := fmt.Sprintf("%s-%s", config.ProjectKey.ValueString(), config.Name.ValueString())
	if len(name) > 32 {
		resp.Diagnostics.AddError(
			"Invalid Attributes Configuration",
			"Combined length of project_key and name (separated by '-') cannot exceed 32 characters",
		)
		return
	}
}
