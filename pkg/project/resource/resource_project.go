package project

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
	validatorfw_string "github.com/jfrog/terraform-provider-shared/validator/fw/string"
	"github.com/samber/lo"
)

const (
	ProjectsUrl           = "/access/api/v1/projects"
	ProjectUrl            = ProjectsUrl + "/{projectKey}"
	MaxStorageInGibibytes = 8589934591
)

var customRoleTypeRegex = regexp.MustCompile(fmt.Sprintf("^%s$", customRoleType))

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

type ProjectResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ProjectResourceModelV1 struct {
	ID                     types.String `tfsdk:"id"`
	Key                    types.String `tfsdk:"key"`
	DisplayName            types.String `tfsdk:"display_name"`
	Description            types.String `tfsdk:"description"`
	AdminPrivileges        types.Set    `tfsdk:"admin_privileges"`
	MaxStorageInGibibytes  types.Int64  `tfsdk:"max_storage_in_gibibytes"`
	SoftLimit              types.Bool   `tfsdk:"block_deployments_on_limit"`
	QuotaEmailNotification types.Bool   `tfsdk:"email_notification"`
	Members                types.Set    `tfsdk:"member"`
	Groups                 types.Set    `tfsdk:"group"`
	Roles                  types.Set    `tfsdk:"role"`
	Repos                  types.Set    `tfsdk:"repos"`
}

type ProjectResourceModelV2 struct {
	ID                     types.String `tfsdk:"id"`
	Key                    types.String `tfsdk:"key"`
	DisplayName            types.String `tfsdk:"display_name"`
	Description            types.String `tfsdk:"description"`
	AdminPrivileges        types.Set    `tfsdk:"admin_privileges"`
	MaxStorageInGibibytes  types.Int64  `tfsdk:"max_storage_in_gibibytes"`
	SoftLimit              types.Bool   `tfsdk:"block_deployments_on_limit"`
	QuotaEmailNotification types.Bool   `tfsdk:"email_notification"`
	Members                types.Set    `tfsdk:"member"`
	Groups                 types.Set    `tfsdk:"group"`
	Roles                  types.Set    `tfsdk:"role"`
	Repos                  types.Set    `tfsdk:"repos"`
	UseProjectRoleResource types.Bool   `tfsdk:"use_project_role_resource"`
}

type ProjectResourceModelV3 struct {
	ID                      types.String `tfsdk:"id"`
	Key                     types.String `tfsdk:"key"`
	DisplayName             types.String `tfsdk:"display_name"`
	Description             types.String `tfsdk:"description"`
	AdminPrivileges         types.Set    `tfsdk:"admin_privileges"`
	MaxStorageInGibibytes   types.Int64  `tfsdk:"max_storage_in_gibibytes"`
	SoftLimit               types.Bool   `tfsdk:"block_deployments_on_limit"`
	QuotaEmailNotification  types.Bool   `tfsdk:"email_notification"`
	Members                 types.Set    `tfsdk:"member"`
	Groups                  types.Set    `tfsdk:"group"`
	Roles                   types.Set    `tfsdk:"role"`
	Repos                   types.Set    `tfsdk:"repos"`
	UseProjectRoleResource  types.Bool   `tfsdk:"use_project_role_resource"`
	UseProjectUserResource  types.Bool   `tfsdk:"use_project_user_resource"`
	UseProjectGroupResource types.Bool   `tfsdk:"use_project_group_resource"`
}

type ProjectResourceModelV4 struct {
	ID                           types.String `tfsdk:"id"`
	Key                          types.String `tfsdk:"key"`
	DisplayName                  types.String `tfsdk:"display_name"`
	Description                  types.String `tfsdk:"description"`
	AdminPrivileges              types.Set    `tfsdk:"admin_privileges"`
	MaxStorageInGibibytes        types.Int64  `tfsdk:"max_storage_in_gibibytes"`
	SoftLimit                    types.Bool   `tfsdk:"block_deployments_on_limit"`
	QuotaEmailNotification       types.Bool   `tfsdk:"email_notification"`
	Members                      types.Set    `tfsdk:"member"`
	Groups                       types.Set    `tfsdk:"group"`
	Roles                        types.Set    `tfsdk:"role"`
	Repos                        types.Set    `tfsdk:"repos"`
	UseProjectRoleResource       types.Bool   `tfsdk:"use_project_role_resource"`
	UseProjectUserResource       types.Bool   `tfsdk:"use_project_user_resource"`
	UseProjectGroupResource      types.Bool   `tfsdk:"use_project_group_resource"`
	UseProjectRepositoryResource types.Bool   `tfsdk:"use_project_repository_resource"`
}

var adminPrivilegesAttrType = map[string]attr.Type{
	"manage_members":   types.BoolType,
	"manage_resources": types.BoolType,
	"index_resources":  types.BoolType,
}

var adminPrivilegesElemType = types.ObjectType{
	AttrTypes: adminPrivilegesAttrType,
}

var memberAttrTypes = map[string]attr.Type{
	"name":  types.StringType,
	"roles": types.SetType{ElemType: types.StringType},
}

var memberElemType = types.ObjectType{
	AttrTypes: memberAttrTypes,
}

var roleAttrTypes = map[string]attr.Type{
	"name":         types.StringType,
	"description":  types.StringType,
	"type":         types.StringType,
	"environments": types.SetType{ElemType: types.StringType},
	"actions":      types.SetType{ElemType: types.StringType},
}

var roleElemType = types.ObjectType{
	AttrTypes: roleAttrTypes,
}

func memberAPIModelsToResourceSet(ctx context.Context, members []MemberAPIModel) (types.Set, diag.Diagnostics) {
	ds := diag.Diagnostics{}

	membersSet := lo.Map(
		members,
		func(member MemberAPIModel, _ int) attr.Value {
			rs, d := types.SetValueFrom(ctx, types.StringType, member.Roles)
			if d.HasError() {
				ds.Append(d...)
			}

			u := map[string]attr.Value{
				"name":  types.StringValue(member.Name),
				"roles": rs,
			}

			m, d := types.ObjectValue(memberAttrTypes, u)
			if d.HasError() {
				ds.Append(d...)
			}
			return m
		},
	)

	return types.SetValue(memberElemType, membersSet)
}

func (r *ProjectResourceModelV4) fromAPIModel(ctx context.Context, apiModel ProjectAPIModel, users, groups []MemberAPIModel, roles []Role, repos []string) diag.Diagnostics {
	ds := diag.Diagnostics{}

	r.ID = types.StringValue(apiModel.Key) // backward compatibility
	r.Key = types.StringValue(apiModel.Key)
	r.DisplayName = types.StringValue(apiModel.DisplayName)

	if len(apiModel.Description) > 0 {
		r.Description = types.StringValue(apiModel.Description)
	}

	r.MaxStorageInGibibytes = types.Int64Value(BytesToGibibytes(apiModel.StorageQuota))
	r.SoftLimit = types.BoolValue(!apiModel.SoftLimit)
	r.QuotaEmailNotification = types.BoolValue(apiModel.QuotaEmailNotification)

	ap := map[string]attr.Value{
		"manage_members":   types.BoolValue(apiModel.AdminPrivileges.ManageMembers),
		"manage_resources": types.BoolValue(apiModel.AdminPrivileges.ManageResources),
		"index_resources":  types.BoolValue(apiModel.AdminPrivileges.IndexResources),
	}
	apObj, d := types.ObjectValue(adminPrivilegesAttrType, ap)
	if d.HasError() {
		ds.Append(d...)
	}
	adminPrivileges, d := types.SetValue(adminPrivilegesElemType, []attr.Value{apObj})
	if d.HasError() {
		ds.Append(d...)
	}
	r.AdminPrivileges = adminPrivileges

	if len(users) > 0 {
		members, d := memberAPIModelsToResourceSet(ctx, users)
		if d.HasError() {
			ds.Append(d...)
			return ds
		}
		r.Members = members
	}

	if len(groups) > 0 {
		members, d := memberAPIModelsToResourceSet(ctx, groups)
		if d.HasError() {
			ds.Append(d...)
			return ds
		}
		r.Groups = members
	}

	if len(roles) > 0 {
		rolesSet := lo.Map(
			roles,
			func(role Role, _ int) attr.Value {
				es, d := types.SetValueFrom(ctx, types.StringType, role.Environments)
				if d.HasError() {
					ds.Append(d...)
				}

				as, d := types.SetValueFrom(ctx, types.StringType, role.Actions)
				if d.HasError() {
					ds.Append(d...)
				}

				r := map[string]attr.Value{
					"name":         types.StringValue(role.Name),
					"description":  types.StringValue(role.Description),
					"type":         types.StringValue(role.Type),
					"environments": es,
					"actions":      as,
				}

				v, d := types.ObjectValue(roleAttrTypes, r)
				if d.HasError() {
					ds.Append(d...)
				}
				return v
			},
		)
		rs, d := types.SetValue(roleElemType, rolesSet)
		if d.HasError() {
			ds.Append(d...)
			return ds
		}
		r.Roles = rs
	}

	if len(repos) > 0 {
		rs, d := types.SetValueFrom(ctx, types.StringType, repos)
		if d.HasError() {
			ds.Append(d...)
			return ds
		}
		r.Repos = rs
	}

	return ds
}

func GibibytesToBytes(bytes int64) int64 {
	if bytes <= -1 {
		return -1
	}

	return bytes * int64(math.Pow(1024, 3))
}

func BytesToGibibytes(bytes int64) int64 {
	if bytes <= -1 {
		return -1
	}

	return int64(bytes / int64(math.Pow(1024, 3)))
}

func resourceMemberToAPIModels(ctx context.Context, members types.Set) ([]MemberAPIModel, diag.Diagnostics) {
	ds := diag.Diagnostics{}

	ms := lo.Map(
		members.Elements(),
		func(elem attr.Value, _ int) MemberAPIModel {
			attrs := elem.(types.Object).Attributes()

			var roles []string
			d := attrs["roles"].(types.Set).ElementsAs(ctx, &roles, false)
			if d.HasError() {
				ds.Append(d...)
			}

			return MemberAPIModel{
				Name:  attrs["name"].(types.String).ValueString(),
				Roles: roles,
			}
		},
	)

	return ms, ds
}

func (r ProjectResourceModelV4) toAPIModel(ctx context.Context, project *ProjectAPIModel, users, groups *[]MemberAPIModel, roles *[]Role, repos *[]string) diag.Diagnostics {
	ds := diag.Diagnostics{}

	proj := ProjectAPIModel{
		Key:                    r.Key.ValueString(),
		DisplayName:            r.DisplayName.ValueString(),
		Description:            r.Description.ValueString(),
		StorageQuota:           GibibytesToBytes(r.MaxStorageInGibibytes.ValueInt64()),
		SoftLimit:              !r.SoftLimit.ValueBool(),
		QuotaEmailNotification: r.QuotaEmailNotification.ValueBool(),
	}

	if !r.AdminPrivileges.IsNull() {
		attrs := r.AdminPrivileges.Elements()[0].(types.Object).Attributes()
		proj.AdminPrivileges.ManageMembers = attrs["manage_members"].(types.Bool).ValueBool()
		proj.AdminPrivileges.ManageResources = attrs["manage_resources"].(types.Bool).ValueBool()
		proj.AdminPrivileges.IndexResources = attrs["index_resources"].(types.Bool).ValueBool()
	}

	*project = proj

	us, d := resourceMemberToAPIModels(ctx, r.Members)
	if d.HasError() {
		ds.Append(d...)
		return ds
	}
	*users = us

	gs, d := resourceMemberToAPIModels(ctx, r.Groups)
	if d.HasError() {
		ds.Append(d...)
		return ds
	}
	*groups = gs

	rs := lo.Map(
		r.Roles.Elements(),
		func(elem attr.Value, _ int) Role {
			attrs := elem.(types.Object).Attributes()

			var es []string
			d := attrs["environments"].(types.Set).ElementsAs(ctx, &es, false)
			if d.HasError() {
				ds.Append(d...)
			}

			var as []string
			d = attrs["actions"].(types.Set).ElementsAs(ctx, &as, false)
			if d.HasError() {
				ds.Append(d...)
			}

			return Role{
				Name:         attrs["name"].(types.String).ValueString(),
				Description:  attrs["description"].(types.String).ValueString(),
				Type:         attrs["type"].(types.String).ValueString(),
				Environments: es,
				Actions:      as,
			}
		},
	)
	*roles = rs

	d = r.Repos.ElementsAs(ctx, repos, false)
	if d.HasError() {
		ds.Append(d...)
		return ds
	}

	return ds
}

type AdminPrivilegesAPIModel struct {
	ManageMembers   bool `json:"manage_members"`
	ManageResources bool `json:"manage_resources"`
	IndexResources  bool `json:"index_resources"`
}

// Project GET {{ host }}/access/api/v1/projects/{{projKey}}/
// GET {{ host }}/artifactory/api/repositories/?project={{projKey}}
type ProjectAPIModel struct {
	Key                    string                  `json:"project_key"`
	DisplayName            string                  `json:"display_name"`
	Description            string                  `json:"description"`
	AdminPrivileges        AdminPrivilegesAPIModel `json:"admin_privileges"`
	StorageQuota           int64                   `json:"storage_quota_bytes"`
	SoftLimit              bool                    `json:"soft_limit"`
	QuotaEmailNotification bool                    `json:"storage_quota_email_notification"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "project"
	r.TypeName = resp.TypeName
}

var schemaV1 = schema.Schema{
	Version: 1,
	Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true,
		},
		"key": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				validatorfw_string.ProjectKey(),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
			Description: "The Project Key is added as a prefix to resources created within a Project. This field is mandatory and supports only 2 - 32 lowercase alphanumeric and hyphen characters. Must begin with a letter. For example: `us1a-test`.",
		},
		"display_name": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 32),
			},
			Description: "Also known as project name on the UI",
		},
		"description": schema.StringAttribute{
			Optional: true,
		},
		"max_storage_in_gibibytes": schema.Int64Attribute{
			Optional: true,
			Computed: true,
			Default:  int64default.StaticInt64(-1),
			Validators: []validator.Int64{
				int64validator.Any(
					int64validator.Between(1, MaxStorageInGibibytes),
					int64validator.OneOf(-1),
				),
			},
			Description: "Storage quota in GiB. Must be 1 or larger. Set to -1 for unlimited storage. This is translated to binary bytes for Artifactory API. So for a 1TB quota, this should be set to 1024 (vs 1000) which will translate to 1099511627776 bytes for the API.",
		},
		"block_deployments_on_limit": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Block deployment of artifacts if storage quota is exceeded.\n\n~>This setting only applies to self-hosted environment. See [Manage Storage Quotas](https://jfrog.com/help/r/jfrog-platform-administration-documentation/manage-storage-quotas).",
		},
		"email_notification": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Alerts will be sent when reaching 75% and 95% of the storage quota. This serves as a notification only and is not a blocker",
		},
		"repos": schema.SetAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
			},
			Description: "(Optional) List of existing repo keys to be assigned to the project. **Note** We *strongly* recommend using this attribute to manage the list of repositories. If you wish to use the alternate method of setting `project_key` attribute in each `artifactory_*_repository` resource in the `artifactory` provider, you will need to use `lifecycle.ignore_changes` in the `project` resource to avoid state drift.\n\n```hcl\nlifecycle {\n\tignore_changes = [\n\t\trepos\n\t]\n}\n```",
		},
	},
	Blocks: map[string]schema.Block{
		"admin_privileges": schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"manage_members": schema.BoolAttribute{
						Required:    true,
						Description: "Allows the Project Admin to manage Platform users/groups as project members with different roles.",
					},
					"manage_resources": schema.BoolAttribute{
						Required:    true,
						Description: "Allows the Project Admin to manage resources - repositories, builds and Pipelines resources on the project level.",
					},
					"index_resources": schema.BoolAttribute{
						Required:    true,
						Description: "Enables a project admin to define the resources to be indexed by Xray",
					},
				},
			},
			Validators: []validator.Set{
				setvalidator.IsRequired(),
				setvalidator.SizeBetween(1, 1),
			},
		},
		"member": schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						Description: "Must be existing Artifactory user",
					},
					"roles": schema.SetAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "List of pre-defined Project or custom roles",
					},
				},
			},
			Description: "Member of the project. Element has one to one mapping with the [JFrog Project Users API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateUserinProject).",
		},
		"group": schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						Description: "Must be existing Artifactory group",
					},
					"roles": schema.SetAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "List of pre-defined Project or custom roles",
					},
				},
			},
			Description: "Project group. Element has one to one mapping with the [JFrog Project Groups API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateGroupinProject)",
		},
		"role": schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 64),
						},
					},
					"description": schema.StringAttribute{
						Optional: true,
					},
					"type": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(customRoleTypeRegex, fmt.Sprintf(`Only "%s" is supported`, customRoleType)),
						},
						Description: fmt.Sprintf(`Type of role. Only "%s" is supported`, customRoleType),
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
			},
			Description: "Project role. Element has one to one mapping with the [JFrog Project Roles API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-AddaNewRole)",
		},
	},
	Description: "Provides an Artifactory project resource. This can be used to create and manage Artifactory project, maintain users/groups/roles/repos.\n\n## Repository Configuration\n\nAfter the project configuration is applied, the repository's attributes `project_key` and `project_environments` would be updated with the project's data. This will generate a state drift in the next Terraform plan/apply for the repository resource. To avoid this, apply `lifecycle.ignore_changes`:\n```hcl\nresource \"artifactory_local_maven_repository\" \"my_maven_releases\" {\n\tkey = \"my-maven-releases\"\n\t...\n\n\tlifecycle {\n\t\tignore_changes = [\n\t\t\tproject_environments,\n\t\t\tproject_key\n\t\t]\n\t}\n}\n```\n~>We strongly recommend using the 'repos' attribute to manage the list of repositories. See below for additional details.",
}

var schemaV2 = schema.Schema{
	Version: 2,
	Attributes: lo.Assign(schemaV1.Attributes, map[string]schema.Attribute{
		"use_project_role_resource": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "When set to true, this resource will ignore the `roles` attributes and allow roles to be managed by `project_role` resource instead. Default to `true`.",
		},
	}),
	Blocks: lo.Assign(schemaV1.Blocks, map[string]schema.Block{
		"role": schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 64),
						},
					},
					"description": schema.StringAttribute{
						Optional: true,
					},
					"type": schema.StringAttribute{
						Optional: true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(customRoleTypeRegex, fmt.Sprintf(`Only "%s" is supported`, customRoleType)),
						},
						Description: fmt.Sprintf(`Type of role. Only "%s" is supported`, customRoleType),
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
			},
			Description:        "Project role. Element has one to one mapping with the [JFrog Project Roles API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-AddaNewRole)",
			DeprecationMessage: "Replaced by `project_role` resource. This should not be used in combination with `project_role` resource. Use `use_project_role_resource` attribute to control which resource manages project roles.",
		},
	}),
}

var schemaV3 = schema.Schema{
	Version: 3,
	Attributes: lo.Assign(schemaV2.Attributes, map[string]schema.Attribute{
		"use_project_user_resource": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "When set to true, this resource will ignore the `member` attributes and allow users to be managed by `project_user` resource instead. Default to `true`.",
		},
		"use_project_group_resource": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "When set to true, this resource will ignore the `group` attributes and allow users to be managed by `project_group` resource instead. Default to `true`.",
		},
	}),
	Blocks: lo.Assign(schemaV2.Blocks, map[string]schema.Block{
		"member": schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						Description: "Must be existing Artifactory user",
					},
					"roles": schema.SetAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "List of pre-defined Project or custom roles",
					},
				},
			},
			Description:        "Member of the project. Element has one to one mapping with the [JFrog Project Users API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateUserinProject).",
			DeprecationMessage: "Replaced by `project_user` resource. This should not be used in combination with `project_user` resource. Use `use_project_user_resource` attribute to control which resource manages project roles.",
		},
		"group": schema.SetNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
						Description: "Must be existing Artifactory group",
					},
					"roles": schema.SetAttribute{
						ElementType: types.StringType,
						Required:    true,
						Description: "List of pre-defined Project or custom roles",
					},
				},
			},
			Description:        "Project group. Element has one to one mapping with the [JFrog Project Groups API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateGroupinProject)",
			DeprecationMessage: "Replaced by `project_group` resource. This should not be used in combination with `project_group` resource. Use `use_project_group_resource` attribute to control which resource manages project roles.",
		},
	}),
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 4,
		Attributes: lo.Assign(schemaV3.Attributes, map[string]schema.Attribute{
			"use_project_repository_resource": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "When set to true, this resource will ignore the `repos` attributes and allow repository to be managed by `project_repository` resource instead. Default to `true`.",
			},
			"repos": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				Description:        "(Optional) List of existing repo keys to be assigned to the project. **Note** We *strongly* recommend using this attribute to manage the list of repositories. If you wish to use the alternate method of setting `project_key` attribute in each `artifactory_*_repository` resource in the `artifactory` provider, you will need to use `lifecycle.ignore_changes` in the `project` resource to avoid state drift.\n\n```hcl\nlifecycle {\n\tignore_changes = [\n\t\trepos\n\t]\n}\n```",
				DeprecationMessage: "Replaced by `project_repository` resource. This should not be used in combination with `project_repository` resource. Use `use_project_repository_resource` attribute to control which resource manages project repositories.",
			},
		}),
		Blocks: schemaV3.Blocks,
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectResourceModelV4

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var project ProjectAPIModel
	var users []MemberAPIModel
	var groups []MemberAPIModel
	var roles []Role
	var repos []string
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &project, &users, &groups, &roles, &repos)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetBody(project).
		SetError(&projectError).
		Post(ProjectsUrl)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
	}
	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, projectError.String())
	}

	// backward compatibility
	plan.ID = types.StringValue(project.Key)

	if !plan.UseProjectRoleResource.ValueBool() {
		_, err = updateRoles(ctx, project.Key, roles, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToCreateResourceError(resp, err.Error())
			return
		}
	}

	if !plan.UseProjectUserResource.ValueBool() {
		_, err = updateMembers(ctx, project.Key, usersMembershipType, users, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToCreateResourceError(resp, err.Error())
			return
		}
	}

	if !plan.UseProjectGroupResource.ValueBool() {
		_, err = updateMembers(ctx, project.Key, groupsMembershipType, groups, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToCreateResourceError(resp, err.Error())
			return
		}
	}

	if !plan.UseProjectRepositoryResource.ValueBool() {
		_, err = updateRepos(ctx, project.Key, repos, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToCreateResourceError(resp, err.Error())
			return
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectResourceModelV4
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var project ProjectAPIModel
	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParam("projectKey", state.Key.ValueString()).
		SetResult(&project).
		SetError(&projectError).
		Get(ProjectUrl)
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

	users := []MemberAPIModel{}
	if !state.UseProjectUserResource.ValueBool() {
		users, err = readMembers(ctx, state.Key.ValueString(), usersMembershipType, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToRefreshResourceError(resp, err.Error())
			return
		}
	}

	groups := []MemberAPIModel{}
	if !state.UseProjectUserResource.ValueBool() {
		groups, err = readMembers(ctx, state.Key.ValueString(), groupsMembershipType, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToRefreshResourceError(resp, err.Error())
			return
		}
	}

	roles := []Role{}
	if !state.UseProjectUserResource.ValueBool() {
		roles, err = readRoles(ctx, state.Key.ValueString(), r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToRefreshResourceError(resp, err.Error())
			return
		}
	}

	repos := []string{}
	if !state.UseProjectUserResource.ValueBool() {
		repos, err = readRepos(ctx, state.Key.ValueString(), r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToRefreshResourceError(resp, err.Error())
			return
		}
	}

	// Convert from the API data model to the Terraform data model
	// and refresh any attribute values.
	resp.Diagnostics.Append(state.fromAPIModel(ctx, project, users, groups, roles, repos)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ProjectResourceModelV4

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var project ProjectAPIModel
	var users []MemberAPIModel
	var groups []MemberAPIModel
	var roles []Role
	var repos []string
	resp.Diagnostics.Append(plan.toAPIModel(ctx, &project, &users, &groups, &roles, &repos)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParam("projectKey", project.Key).
		SetBody(project).
		SetError(&projectError).
		Put(ProjectUrl)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
	}
	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, projectError.String())
	}

	// backward compatibility
	plan.ID = types.StringValue(project.Key)

	if !plan.UseProjectRoleResource.ValueBool() {
		_, err = updateRoles(ctx, project.Key, roles, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToUpdateResourceError(resp, err.Error())
			return
		}
	}

	if !plan.UseProjectUserResource.ValueBool() {
		_, err = updateMembers(ctx, project.Key, usersMembershipType, users, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToUpdateResourceError(resp, err.Error())
			return
		}
	}

	if !plan.UseProjectGroupResource.ValueBool() {
		_, err = updateMembers(ctx, project.Key, groupsMembershipType, groups, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToUpdateResourceError(resp, err.Error())
			return
		}
	}

	if !plan.UseProjectRepositoryResource.ValueBool() {
		_, err = updateRepos(ctx, project.Key, repos, r.ProviderData.Client)
		if err != nil {
			utilfw.UnableToUpdateResourceError(resp, err.Error())
			return
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectResourceModelV4

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var repos []string
	resp.Diagnostics.Append(state.Repos.ElementsAs(ctx, &repos, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteErr := deleteRepos(ctx, repos, r.ProviderData.Client)
	if deleteErr != nil {
		utilfw.UnableToDeleteResourceError(resp, fmt.Sprintf("failed to delete repos for project: %s", deleteErr))
		return
	}

	var projectError ProjectErrorsResponse
	response, err := r.ProviderData.Client.R().
		SetPathParam("projectKey", state.Key.ValueString()).
		SetError(&projectError).
		AddRetryCondition(
			func(r *resty.Response, _ error) bool {
				return r.StatusCode() == http.StatusBadRequest &&
					strings.Contains(r.String(), "project containing resources can't be removed")
			},
		).
		Delete(ProjectUrl)
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
func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("key"), req, resp)
}

func (r *ProjectResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade implementation from 1 (prior state version) to 4 (Schema.Version)
		1: {
			PriorSchema: &schemaV1,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) { /* ... */
				var priorStateData ProjectResourceModelV1

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := ProjectResourceModelV4{
					ID:                     priorStateData.ID,
					Key:                    priorStateData.Key,
					DisplayName:            priorStateData.DisplayName,
					Description:            priorStateData.Description,
					AdminPrivileges:        priorStateData.AdminPrivileges,
					MaxStorageInGibibytes:  priorStateData.MaxStorageInGibibytes,
					SoftLimit:              priorStateData.SoftLimit,
					QuotaEmailNotification: priorStateData.QuotaEmailNotification,
					Members:                priorStateData.Members,
					Groups:                 priorStateData.Groups,
					Roles:                  priorStateData.Roles,
					Repos:                  priorStateData.Repos,

					UseProjectRoleResource:       types.BoolValue(false),
					UseProjectUserResource:       types.BoolValue(false),
					UseProjectGroupResource:      types.BoolValue(false),
					UseProjectRepositoryResource: types.BoolValue(false),
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
		// State upgrade implementation from 2 (prior state version) to 4 (Schema.Version)
		2: {
			PriorSchema: &schemaV2,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) { /* ... */
				var priorStateData ProjectResourceModelV2

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := ProjectResourceModelV4{
					ID:                     priorStateData.ID,
					Key:                    priorStateData.Key,
					DisplayName:            priorStateData.DisplayName,
					Description:            priorStateData.Description,
					AdminPrivileges:        priorStateData.AdminPrivileges,
					MaxStorageInGibibytes:  priorStateData.MaxStorageInGibibytes,
					SoftLimit:              priorStateData.SoftLimit,
					QuotaEmailNotification: priorStateData.QuotaEmailNotification,
					Members:                priorStateData.Members,
					Groups:                 priorStateData.Groups,
					Roles:                  priorStateData.Roles,
					Repos:                  priorStateData.Repos,
					UseProjectRoleResource: priorStateData.UseProjectRoleResource,

					UseProjectUserResource:       types.BoolValue(false),
					UseProjectGroupResource:      types.BoolValue(false),
					UseProjectRepositoryResource: types.BoolValue(false),
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
		// State upgrade implementation from 2 (prior state version) to 4 (Schema.Version)
		3: {
			PriorSchema: &schemaV3,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) { /* ... */
				var priorStateData ProjectResourceModelV3

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgradedStateData := ProjectResourceModelV4{
					ID:                      priorStateData.ID,
					Key:                     priorStateData.Key,
					DisplayName:             priorStateData.DisplayName,
					Description:             priorStateData.Description,
					AdminPrivileges:         priorStateData.AdminPrivileges,
					MaxStorageInGibibytes:   priorStateData.MaxStorageInGibibytes,
					SoftLimit:               priorStateData.SoftLimit,
					QuotaEmailNotification:  priorStateData.QuotaEmailNotification,
					Members:                 priorStateData.Members,
					Groups:                  priorStateData.Groups,
					Roles:                   priorStateData.Roles,
					Repos:                   priorStateData.Repos,
					UseProjectRoleResource:  priorStateData.UseProjectRoleResource,
					UseProjectUserResource:  priorStateData.UseProjectUserResource,
					UseProjectGroupResource: priorStateData.UseProjectGroupResource,

					UseProjectRepositoryResource: types.BoolValue(false),
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
	}
}
