package project

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/jfrog/terraform-provider-shared/util/sdk"
	"github.com/jfrog/terraform-provider-shared/validator"
)

type AdminPrivileges struct {
	ManageMembers   bool `json:"manage_members"`
	ManageResources bool `json:"manage_resources"`
	IndexResources  bool `json:"index_resources"`
}

// Project GET {{ host }}/access/api/v1/projects/{{prjKey}}/
// GET {{ host }}/artifactory/api/repositories/?prjKey={{prjKey}}
type Project struct {
	Key                    string          `json:"project_key"`
	DisplayName            string          `json:"display_name"`
	Description            string          `json:"description"`
	AdminPrivileges        AdminPrivileges `json:"admin_privileges"`
	StorageQuota           int64           `json:"storage_quota_bytes"`
	SoftLimit              bool            `json:"soft_limit"`
	QuotaEmailNotification bool            `json:"storage_quota_email_notification"`
}

func (p Project) Id() string {
	return p.Key
}

const projectsUrl = "/access/api/v1/projects"
const projectUrl = projectsUrl + "/{projectKey}"
const maxStorageInGibibytes = 8589934591

var customRoleTypeRegex = regexp.MustCompile(fmt.Sprintf("^%s$", customRoleType))

func projectResource() *schema.Resource {
	var projectSchema = map[string]*schema.Schema{
		"key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.ProjectKey,
			Description:      "The Project Key is added as a prefix to resources created within a Project. This field is mandatory and supports only 2 - 20 lowercase alphanumeric and hyphen characters. Must begin with a letter. For example: `us1a-test`.",
		},
		"display_name": {
			Required: true,
			Type:     schema.TypeString,
			ValidateDiagFunc: validation.ToDiagFunc(validation.All(
				validation.StringIsNotEmpty,
				maxLength(32),
			)),
			Description: "Also known as project name on the UI",
		},
		"description": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"admin_privileges": {
			Type:     schema.TypeSet,
			Required: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"manage_members": {
						Type:        schema.TypeBool,
						Required:    true,
						Description: "Allows the Project Admin to manage Platform users/groups as project members with different roles.",
					},
					"manage_resources": {
						Type:        schema.TypeBool,
						Required:    true,
						Description: "Allows the Project Admin to manage resources - repositories, builds and Pipelines resources on the project level.",
					},
					"index_resources": {
						Type:        schema.TypeBool,
						Required:    true,
						Description: "Enables a project admin to define the resources to be indexed by Xray",
					},
				},
			},
		},
		"max_storage_in_gibibytes": {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  -1,
			ValidateDiagFunc: validation.ToDiagFunc(
				validation.Any(
					int64Between(1, maxStorageInGibibytes),
					validation.IntInSlice([]int{-1}),
				),
			),
			DiffSuppressFunc: func(key, old, new string, d *schema.ResourceData) bool {
				oldVal, err := strconv.Atoi(old)
				if err != nil {
					return false
				}
				newVal, err := strconv.Atoi(new)
				if err != nil {
					return false
				}
				// convert to bytes. The API says bytes, but the UI only allows GB (which makes more sense)
				oldVal = oldVal * 1024 * 1024 * 1024
				newVal = newVal * 1024 * 1024 * 1024
				return newVal == oldVal
			},
			Description: "Storage quota in GiB. Must be 1 or larger. Set to -1 for unlimited storage. This is translated to binary bytes for Artifactory API. So for a 1TB quota, this should be set to 1024 (vs 1000) which will translate to 1099511627776 bytes for the API.",
		},
		"block_deployments_on_limit": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Block deployment of artifacts if storage quota is exceeded.\n\n~>This setting only applies to self-hosted environment. See [Manage Storage Quotas](https://jfrog.com/help/r/jfrog-platform-administration-documentation/manage-storage-quotas).",
		},
		"email_notification": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Alerts will be sent when reaching 75% and 95% of the storage quota. This serves as a notification only and is not a blocker",
		},
		"member": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
						Description:      "Must be existing Artifactory user",
					},
					"roles": {
						Type:        schema.TypeSet,
						Required:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: "List of pre-defined Project or custom roles",
					},
				},
			},
			Description: "Member of the project. Element has one to one mapping with the [JFrog Project Users API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateUserinProject).",
		},
		"group": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
						Description:      "Must be existing Artifactory group",
					},
					"roles": {
						Type:        schema.TypeSet,
						Required:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: "List of pre-defined Project or custom roles",
					},
				},
			},
			Description: "Project group. Element has one to one mapping with the [JFrog Project Groups API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateGroupinProject)",
		},
		"role": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:     schema.TypeString,
						Required: true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.All(
							validation.StringIsNotEmpty,
							maxLength(64),
						)),
					},
					"description": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"type": {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(customRoleTypeRegex, fmt.Sprintf(`Only "%s" is supported`, customRoleType))),
						Description:      fmt.Sprintf(`Type of role. Only "%s" is supported`, customRoleType),
					},
					"environments": {
						Type:        schema.TypeSet,
						Required:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: fmt.Sprintf("A repository can be available in different environments. Members with roles defined in the set environment will have access to the repository. List of pre-defined environments (%s)", strings.Join(validRoleEnvironments, ", ")),
					},
					"actions": {
						Type:        schema.TypeSet,
						Required:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: fmt.Sprintf("List of pre-defined actions (%s)", strings.Join(validRoleActions, ", ")),
					},
				},
			},
			Description: "Project role. Element has one to one mapping with the [JFrog Project Roles API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-AddaNewRole)",
		},
		"repos": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			MinItems:    0,
			Description: "(Optional) List of existing repo keys to be assigned to the project. **Note** We *strongly* recommend using this attribute to manage the list of repositories. If you wish to use the alternate method of setting `project_key` attribute in each `artifactory_*_repository` resource in the `artifactory` provider, you will need to use `lifecycle.ignore_changes` in the `project` resource to avoid state drift.\n\n```hcl\nlifecycle {\n\tignore_changes = [\n\t\trepos\n\t]\n}\n```",
		},
	}

	var projectSchemaV2 = sdk.MergeMaps(
		projectSchema,
		map[string]*schema.Schema{
			"role": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.All(
								validation.StringIsNotEmpty,
								maxLength(64),
							)),
						},
						"description": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(customRoleTypeRegex, fmt.Sprintf(`Only "%s" is supported`, customRoleType))),
							Description:      fmt.Sprintf(`Type of role. Only "%s" is supported`, customRoleType),
						},
						"environments": {
							Type:        schema.TypeSet,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: fmt.Sprintf("A repository can be available in different environments. Members with roles defined in the set environment will have access to the repository. List of pre-defined environments (%s)", strings.Join(validRoleEnvironments, ", ")),
						},
						"actions": {
							Type:        schema.TypeSet,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: fmt.Sprintf("List of pre-defined actions (%s)", strings.Join(validRoleActions, ", ")),
						},
					},
				},
				Description: "Project role. Element has one to one mapping with the [JFrog Project Roles API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-AddaNewRole)",
				Deprecated:  "Replaced by `project_role` resource. This should not be used in combination with `project_role` resource. Use `use_project_role_resource` attribute to control which resource manages project roles.",
			},
			"use_project_role_resource": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When set to true, this resource will ignore the `roles` attributes and allow roles to be managed by `project_role` resource instead. Default to `true`.",
			},
		},
	)

	var projectSchemaV3 = sdk.MergeMaps(
		projectSchemaV2,
		map[string]*schema.Schema{
			"member": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
							Description:      "Must be existing Artifactory user",
						},
						"roles": {
							Type:        schema.TypeSet,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "List of pre-defined Project or custom roles",
						},
					},
				},
				Description: "Member of the project. Element has one to one mapping with the [JFrog Project Users API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateUserinProject).",
				Deprecated:  "Replaced by `project_user` resource. This should not be used in combination with `project_user` resource. Use `use_project_user_resource` attribute to control which resource manages project roles.",
			},
			"use_project_user_resource": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When set to true, this resource will ignore the `member` attributes and allow users to be managed by `project_user` resource instead. Default to `true`.",
			},
			"group": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
							Description:      "Must be existing Artifactory group",
						},
						"roles": {
							Type:        schema.TypeSet,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "List of pre-defined Project or custom roles",
						},
					},
				},
				Description: "Project group. Element has one to one mapping with the [JFrog Project Groups API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-UpdateGroupinProject)",
				Deprecated:  "Replaced by `project_group` resource. This should not be used in combination with `project_group` resource. Use `use_project_group_resource` attribute to control which resource manages project roles.",
			},
			"use_project_group_resource": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When set to true, this resource will ignore the `group` attributes and allow users to be managed by `project_group` resource instead. Default to `true`.",
			},
		},
	)

	var projectSchemaV4 = sdk.MergeMaps(
		projectSchemaV3,
		map[string]*schema.Schema{
			"use_project_repository_resource": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "When set to true, this resource will ignore the `repos` attributes and allow repository to be managed by `project_repository` resource instead. Default to `true`.",
			},
		},
	)

	var unpackProject = func(data *schema.ResourceData) (Project, Membership, Membership, []Role, []RepoKey, error) {
		d := &sdk.ResourceData{ResourceData: data}

		project := Project{
			Key:                    d.GetString("key", false),
			DisplayName:            d.GetString("display_name", false),
			Description:            d.GetString("description", false),
			StorageQuota:           GibibytesToBytes(d.GetInt("max_storage_in_gibibytes", false)),
			SoftLimit:              !d.GetBool("block_deployments_on_limit", false),
			QuotaEmailNotification: d.GetBool("email_notification", false),
		}

		if v, ok := d.GetOk("admin_privileges"); ok {
			privileges := v.(*schema.Set).List()
			if len(privileges) == 1 {
				adminPrivileges := AdminPrivileges{}

				id := privileges[0].(map[string]interface{})

				adminPrivileges.ManageMembers = id["manage_members"].(bool)
				adminPrivileges.ManageResources = id["manage_resources"].(bool)
				adminPrivileges.IndexResources = id["index_resources"].(bool)

				project.AdminPrivileges = adminPrivileges
			}
		}

		users := unpackMembers(data, "member")
		groups := unpackMembers(data, "group")

		roles := unpackRoles(data)
		repos := unpackRepos(data)

		return project, users, groups, roles, repos, nil
	}

	var packProject = func(ctx context.Context, d *schema.ResourceData, project Project, users []Member, groups []Member, roles []Role, repos []RepoKey) diag.Diagnostics {
		var errors []error
		setValue := sdk.MkLens(d)

		setValue("key", project.Key)
		setValue("display_name", project.DisplayName)
		setValue("description", project.Description)
		setValue("max_storage_in_gibibytes", BytesToGibibytes(project.StorageQuota))
		setValue("block_deployments_on_limit", !project.SoftLimit)
		setValue("email_notification", project.QuotaEmailNotification)
		errors = setValue("admin_privileges", []interface{}{
			map[string]bool{
				"manage_members":   project.AdminPrivileges.ManageMembers,
				"manage_resources": project.AdminPrivileges.ManageResources,
				"index_resources":  project.AdminPrivileges.IndexResources,
			},
		})

		if len(users) > 0 {
			errors = packMembers(ctx, d, "member", users)
		}

		if len(groups) > 0 {
			errors = packMembers(ctx, d, "group", groups)
		}

		if len(roles) > 0 {
			errors = packRoles(ctx, d, roles)
		}

		if len(repos) > 0 {
			errors = packRepos(ctx, d, repos)
		}

		if len(errors) > 0 {
			return diag.Errorf("failed to pack project %q", errors)
		}

		return nil
	}

	var readProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		project := Project{}

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParam("projectKey", data.Id()).
			SetResult(&project).
			Get(projectUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		users := []Member{}
		useProjectUserResource := data.Get("use_project_user_resource").(bool)
		if !useProjectUserResource {
			users, err = readMembers(ctx, data.Id(), usersMembershipType, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		groups := []Member{}
		useProjectGroupResource := data.Get("use_project_group_resource").(bool)
		if !useProjectGroupResource {
			groups, err = readMembers(ctx, data.Id(), groupssMembershipType, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		roles := []Role{}
		useProjectRoleResource := data.Get("use_project_role_resource").(bool)
		if !useProjectRoleResource {
			roles, err = readRoles(ctx, data.Id(), m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		repos := []RepoKey{}
		useProjectRepositoryResource := data.Get("use_project_repository_resource").(bool)
		if !useProjectRepositoryResource {
			repos, err = readRepos(ctx, data.Id(), m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		return packProject(ctx, data, project, users, groups, roles, repos)
	}

	var createProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		tflog.Debug(ctx, "createProject")
		tflog.Trace(ctx, fmt.Sprintf("%+v\n", data))

		project, users, groups, roles, repos, err := unpackProject(data)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = m.(util.ProvderMetadata).Client.R().
			SetBody(project).
			Post(projectsUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(project.Id())

		// Role should be updated first before members or groups as they may depend on roles defined by the users
		useProjectRoleResource := data.Get("use_project_role_resource").(bool)
		if !useProjectRoleResource {
			_, err = updateRoles(ctx, data.Id(), roles, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		useProjectUserResource := data.Get("use_project_user_resource").(bool)
		if !useProjectUserResource {
			_, err = updateMembers(ctx, data.Id(), usersMembershipType, users, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		useProjectGroupResource := data.Get("use_project_group_resource").(bool)
		if !useProjectGroupResource {
			_, err = updateMembers(ctx, data.Id(), groupssMembershipType, groups, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		useProjectRepositoryResource := data.Get("use_project_repository_resource").(bool)
		if !useProjectRepositoryResource {
			_, err = updateRepos(ctx, data.Id(), repos, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		return readProject(ctx, data, m)
	}

	var updateProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		tflog.Debug(ctx, "updateProject")
		tflog.Trace(ctx, fmt.Sprintf("%+v\n", data))

		project, users, groups, roles, repos, err := unpackProject(data)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = m.(util.ProvderMetadata).Client.R().
			SetPathParam("projectKey", data.Id()).
			SetBody(project).
			Put(projectUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(project.Id())

		// Role should be updated first before members or groups as they may depend on roles defined by the users
		useProjectRoleResource := data.Get("use_project_role_resource").(bool)
		if !useProjectRoleResource {
			_, err = updateRoles(ctx, data.Id(), roles, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		useProjectUserResource := data.Get("use_project_user_resource").(bool)
		if !useProjectUserResource {
			_, err = updateMembers(ctx, data.Id(), usersMembershipType, users, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		useProjectGroupResource := data.Get("use_project_group_resource").(bool)
		if !useProjectGroupResource {
			_, err = updateMembers(ctx, data.Id(), groupssMembershipType, groups, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		useProjectRepositoryResource := data.Get("use_project_repository_resource").(bool)
		if !useProjectRepositoryResource {
			_, err = updateRepos(ctx, data.Id(), repos, m)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		return readProject(ctx, data, m)
	}

	var deleteProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		tflog.Debug(ctx, "deleteProject")
		tflog.Trace(ctx, fmt.Sprintf("%+v\n", data))

		_, _, _, _, repos, err := unpackProject(data)
		if err != nil {
			return diag.FromErr(err)
		}

		deleteErr := deleteRepos(ctx, repos, m)
		if deleteErr != nil {
			return diag.FromErr(fmt.Errorf("failed to delete repos for project: %s", deleteErr))
		}

		req := m.(util.ProvderMetadata).Client.R()
		req.AddRetryCondition(
			func(r *resty.Response, _ error) bool {
				return r.StatusCode() == http.StatusBadRequest &&
					strings.Contains(r.String(), "project containing resources can't be removed")
			},
		)

		resp, err := req.
			SetPathParam("projectKey", data.Id()).
			Delete(projectUrl)

		if err != nil {
			if resp.StatusCode() == http.StatusNotFound {
				data.SetId("")
			}
			return diag.FromErr(err)
		}

		return nil
	}

	var resourceV1 = func() *schema.Resource {
		return &schema.Resource{
			Schema: projectSchema,
		}
	}

	var resourceV2 = func() *schema.Resource {
		return &schema.Resource{
			Schema: projectSchemaV2,
		}
	}

	var resourceV3 = func() *schema.Resource {
		return &schema.Resource{
			Schema: projectSchemaV3,
		}
	}

	var resourceStateUpgradeV1 = func(ctx context.Context, rawState map[string]any, meta any) (map[string]any, error) {
		// set use_project_role_resource to false for existing state so the resource will continue
		// using `roles` attribute until explicitly set to true
		rawState["use_project_role_resource"] = false
		return rawState, nil
	}

	var resourceStateUpgradeV2 = func(ctx context.Context, rawState map[string]any, meta any) (map[string]any, error) {
		// like in v1 where the project_role was introduced, just for project_user and project_group
		rawState["use_project_user_resource"] = false
		rawState["use_project_group_resource"] = false
		return rawState, nil
	}

	var resourceStateUpgradeV3 = func(ctx context.Context, rawState map[string]any, meta any) (map[string]any, error) {
		rawState["use_project_repository_resource"] = false
		return rawState, nil
	}

	return &schema.Resource{
		CreateContext: createProject,
		ReadContext:   readProject,
		UpdateContext: updateProject,
		DeleteContext: deleteProject,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema:        projectSchemaV4,
		SchemaVersion: 4,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceV1().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceStateUpgradeV1,
				Version: 1,
			},
			{
				Type:    resourceV2().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceStateUpgradeV2,
				Version: 2,
			},
			{
				Type:    resourceV3().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceStateUpgradeV3,
				Version: 3,
			},
		},
		Description: "Provides an Artifactory project resource. This can be used to create and manage Artifactory project, maintain users/groups/roles/repos.\n\n## Repository Configuration\n\nAfter the project configuration is applied, the repository's attributes `project_key` and `project_environments` would be updated with the project's data. This will generate a state drift in the next Terraform plan/apply for the repository resource. To avoid this, apply `lifecycle.ignore_changes`:\n```hcl\nresource \"artifactory_local_maven_repository\" \"my_maven_releases\" {\n\tkey = \"my-maven-releases\"\n\t...\n\n\tlifecycle {\n\t\tignore_changes = [\n\t\t\tproject_environments,\n\t\t\tproject_key\n\t\t]\n\t}\n}\n```\n~>We strongly recommend using the 'repos' attribute to manage the list of repositories. See below for additional details.",
	}
}
