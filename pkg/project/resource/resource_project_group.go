package project

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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

const ProjectGroupsUrl = "access/api/v1/projects/{projectKey}/groups/{name}"

func NewProjectGroupResource() resource.Resource {
	return &ProjectGroupResource{
		TypeName: "project_group",
	}
}

type ProjectGroupResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ProjectGroupResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	ProjectKey types.String `tfsdk:"project_key"`
	Roles      types.Set    `tfsdk:"roles"`
}

type ProjectGroupAPIModel struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

func (r *ProjectGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ProjectGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				Description: "The name of an artifactory group.",
			},
			"project_key": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validatorfw_string.ProjectKey(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The key of the project to which the group should be assigned to.",
			},
			"roles": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				Description: "List of pre-defined Project or custom roles. Must have at least 1 role, e.g. 'Viewer'",
			},
		},
		Description: "Add a group as project member. Element has one to one mapping with the [JFrog Project Groups API](https://jfrog.com/help/r/jfrog-rest-apis/update-group-in-project). Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.",
	}
}

func (r *ProjectGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ProjectGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectGroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()

	var roles []string
	resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := ProjectGroupAPIModel{
		Name:  plan.Name.ValueString(),
		Roles: roles,
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"name":       plan.Name.ValueString(),
		}).
		SetBody(group).
		SetError(&projectError).
		Put(ProjectGroupsUrl)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
	}
	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, projectError.String())
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", projectKey, group.Name))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectGroupResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := state.ProjectKey.ValueString()

	var group ProjectGroupAPIModel
	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"name":       state.Name.ValueString(),
		}).
		SetResult(&group).
		SetError(&projectError).
		Get(ProjectGroupsUrl)
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

	state.ID = types.StringValue(fmt.Sprintf("%s:%s", projectKey, group.Name))
	state.Name = types.StringValue(group.Name)
	state.ProjectKey = types.StringValue(projectKey)
	roles, ds := types.SetValueFrom(ctx, types.StringType, group.Roles)
	if ds.HasError() {
		resp.Diagnostics.Append(ds...)
		return
	}
	state.Roles = roles

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectGroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()

	var roles []string
	resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := ProjectGroupAPIModel{
		Name:  plan.Name.ValueString(),
		Roles: roles,
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"name":       plan.Name.ValueString(),
		}).
		SetBody(group).
		SetError(&projectError).
		Put(ProjectGroupsUrl)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
	}
	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, projectError.String())
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", projectKey, group.Name))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectGroupResourceModel

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
			"name":       state.Name.ValueString(),
		}).
		SetError(&projectError).
		Delete(ProjectGroupsUrl)
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
func (r *ProjectGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected project_key:name",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}
