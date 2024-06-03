package project

import (
	"context"
	"fmt"
	"net/http"
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
)

const ProjectRolesUrl = ProjectUrl + "/roles"
const ProjectRoleUrl = ProjectRolesUrl + "/{roleName}"

const customRoleType = "CUSTOM"

var validRoleEnvironments = []string{
	"DEV",
	"PROD",
}

var validRoleActions = []string{
	"READ_REPOSITORY",
	"ANNOTATE_REPOSITORY",
	"DEPLOY_CACHE_REPOSITORY",
	"DELETE_OVERWRITE_REPOSITORY",
	"MANAGE_XRAY_MD_REPOSITORY",
	"READ_RELEASE_BUNDLE",
	"ANNOTATE_RELEASE_BUNDLE",
	"CREATE_RELEASE_BUNDLE",
	"DISTRIBUTE_RELEASE_BUNDLE",
	"DELETE_RELEASE_BUNDLE",
	"MANAGE_XRAY_MD_RELEASE_BUNDLE",
	"READ_BUILD",
	"ANNOTATE_BUILD",
	"DEPLOY_BUILD",
	"DELETE_BUILD",
	"MANAGE_XRAY_MD_BUILD",
	"READ_SOURCES_PIPELINE",
	"TRIGGER_PIPELINE",
	"READ_INTEGRATIONS_PIPELINE",
	"READ_POOLS_PIPELINE",
	"MANAGE_INTEGRATIONS_PIPELINE",
	"MANAGE_SOURCES_PIPELINE",
	"MANAGE_POOLS_PIPELINE",
	"TRIGGER_SECURITY",
	"ISSUES_SECURITY",
	"LICENCES_SECURITY",
	"REPORTS_SECURITY",
	"WATCHES_SECURITY",
	"POLICIES_SECURITY",
	"RULES_SECURITY",
	"MANAGE_MEMBERS",
	"MANAGE_RESOURCES",
}

func NewProjectRoleResource() resource.Resource {
	return &ProjectRoleResource{}
}

type ProjectRoleResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ProjectRoleResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	ProjectKey   types.String `tfsdk:"project_key"`
	Environments types.Set    `tfsdk:"environments"`
	Actions      types.Set    `tfsdk:"actions"`
}

type ProjectRoleAPIModel struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Environments []string `json:"environments"`
	Actions      []string `json:"actions"`
}

func (r *ProjectRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
	r.TypeName = resp.TypeName
}

func (r *ProjectRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(customRoleTypeRegex, fmt.Sprintf(`Only "%s" is supported`, customRoleType)),
				},
				Description: fmt.Sprintf(`Type of role. Only "%s" is supported`, customRoleType),
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
			"environments": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: fmt.Sprintf("A repository can be available in different environments. Members with roles defined in the set environment will have access to the repository. List of pre-defined environments (%s)", strings.Join(validRoleEnvironments, ", ")),
			},
			"actions": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: fmt.Sprintf("List of pre-defined actions (%s)", strings.Join(validRoleActions, ", ")),
			},
		},
		Description: "Create a project role. Element has one to one mapping with the [JFrog Project Roles API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-AddaNewRole). Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.",
	}
}

func (r *ProjectRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ProjectRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectRoleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()

	var environments []string
	resp.Diagnostics.Append(plan.Environments.ElementsAs(ctx, &environments, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var actions []string
	resp.Diagnostics.Append(plan.Actions.ElementsAs(ctx, &actions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := ProjectRoleAPIModel{
		Name:         plan.Name.ValueString(),
		Type:         plan.Type.ValueString(),
		Environments: environments,
		Actions:      actions,
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParam("projectKey", projectKey).
		SetBody(role).
		SetError(&projectError).
		Post(ProjectRolesUrl)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
	}
	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, projectError.String())
	}

	plan.ID = types.StringValue(role.Name)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectRoleResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := state.ProjectKey.ValueString()

	var role ProjectRoleAPIModel
	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   state.Name.ValueString(),
		}).
		SetResult(&role).
		SetError(&projectError).
		Get(ProjectRoleUrl)
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, projectError.String())
		return
	}

	state.ID = types.StringValue(role.Name)
	state.Name = types.StringValue(role.Name)
	state.Type = types.StringValue(role.Type)
	state.ProjectKey = types.StringValue(projectKey)

	environments, ds := types.SetValueFrom(ctx, types.StringType, role.Environments)
	if ds.HasError() {
		resp.Diagnostics.Append(ds...)
		return
	}
	state.Environments = environments

	actions, ds := types.SetValueFrom(ctx, types.StringType, role.Actions)
	if ds.HasError() {
		resp.Diagnostics.Append(ds...)
		return
	}
	state.Actions = actions

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectRoleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()

	var environments []string
	resp.Diagnostics.Append(plan.Environments.ElementsAs(ctx, &environments, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var actions []string
	resp.Diagnostics.Append(plan.Actions.ElementsAs(ctx, &actions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := ProjectRoleAPIModel{
		Name:         plan.Name.ValueString(),
		Type:         plan.Type.ValueString(),
		Environments: environments,
		Actions:      actions,
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   role.Name,
		}).
		SetBody(role).
		SetError(&projectError).
		Put(ProjectRoleUrl)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
	}
	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, projectError.String())
	}

	plan.ID = types.StringValue(role.Name)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectRoleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := state.ProjectKey.ValueString()

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   state.Name.ValueString(),
		}).
		SetError(&projectError).
		Delete(ProjectRoleUrl)
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
func (r *ProjectRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected project_key:role_name",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}
