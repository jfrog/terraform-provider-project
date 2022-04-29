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
	"github.com/jfrog/terraform-provider-shared/validator"
	"golang.org/x/sync/errgroup"
)

type AdminPrivileges struct {
	ManageMembers   bool `json:"manage_members"`
	ManageResources bool `json:"manage_resources"`
	IndexResources  bool `json:"index_resources"`
}

// Project GET {{ host }}/access/api/v1/projects/{{prjKey}}/
//GET {{ host }}/artifactory/api/repositories/?prjKey={{prjKey}}
type Project struct {
	Key                    string          `json:"project_key"`
	DisplayName            string          `json:"display_name"`
	Description            string          `json:"description"`
	AdminPrivileges        AdminPrivileges `json:"admin_privileges"`
	StorageQuota           int             `json:"storage_quota_bytes"`
	SoftLimit              bool            `json:"soft_limit"`
	QuotaEmailNotification bool            `json:"storage_quota_email_notification"`
}

func (p Project) Id() string {
	return p.Key
}

const projectsUrl = "/access/api/v1/projects"
const projectUrl = projectsUrl + "/{projectKey}"

func verifyProject(id string, request *resty.Request) (*resty.Response, error) {
	return request.Head(projectsUrl + id)
}

var customRoleTypeRegex = regexp.MustCompile(fmt.Sprintf("^%s$", customRoleType))

func projectResource() *schema.Resource {

	var projectSchema = map[string]*schema.Schema{
		"key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.ProjectKey,
			Description:      "The Project Key is added as a prefix to resources created within a Project. This field is mandatory and supports only 3 - 6 lowercase alphanumeric characters. Must begin with a letter. For example: us1a.",
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
					validation.IntAtLeast(1),
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
			Description: "Storage quota in GiB. Must be 1 or larger. Set to -1 for unlimited storage. This is translated to binary bytes for Artifactory API. So for 1TB quota, this should be set to 1024 (vs 1000) which will translate to 1099511627776 bytes for the API.",
		},
		"block_deployments_on_limit": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Block artifacts deployment if storage quota is exceeded.",
		},
		"email_notification": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Alerts will be sent when reaching 75% and 95% of the storage quota. Serves as a notification only and is not a blocker",
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
			MinItems: 0,
			MaxItems: func() int {
				if isOverride := getBoolEnvVar("REPO_LIMIT_OVERRIDE", false); isOverride {
					return 2147483647
				}
				return 100
			}(),
			Description: "(Optional) List of existing repo keys to be assigned to the project.",
		},
	}

	var unpackProject = func(data *schema.ResourceData) (Project, Membership, Membership, []Role, []RepoKey, error) {
		d := &util.ResourceData{data}

		project := Project{
			Key:                    d.GetString("key", false),
			DisplayName:            d.GetString("display_name", false),
			Description:            d.GetString("description", false),
			StorageQuota:           GibibytesToBytes(d.GetInt("max_storage_in_gibibytes", false)),
			SoftLimit:              d.GetBool("block_deployments_on_limit", false),
			QuotaEmailNotification: d.GetBool("email_notification", false),
		}

		if v, ok := d.GetOkExists("admin_privileges"); ok {
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
		setValue := util.MkLens(d)

		setValue("key", project.Key)
		setValue("display_name", project.DisplayName)
		setValue("description", project.Description)
		setValue("max_storage_in_gibibytes", BytesToGibibytes(project.StorageQuota))
		setValue("block_deployments_on_limit", project.SoftLimit)
		errors = setValue("email_notification", project.QuotaEmailNotification)
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

		_, err := m.(*resty.Client).R().
			SetPathParam("projectKey", data.Id()).
			SetResult(&project).
			Get(projectUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		users, err := readMembers(ctx, data.Id(), usersMembershipType, m)
		if err != nil {
			return diag.FromErr(err)
		}

		groups, err := readMembers(ctx, data.Id(), groupssMembershipType, m)
		if err != nil {
			return diag.FromErr(err)
		}

		roles, err := readRoles(ctx, data.Id(), m)
		if err != nil {
			return diag.FromErr(err)
		}

		repos, err := readRepos(ctx, data.Id(), m)
		if err != nil {
			return diag.FromErr(err)
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

		_, err = m.(*resty.Client).R().SetBody(project).Post(projectsUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(project.Id())

		// Role should be updated first before members or groups as they may depend on roles defined by the users
		_, err = updateRoles(ctx, data.Id(), roles, m)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = updateMembers(ctx, data.Id(), usersMembershipType, users, m)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = updateMembers(ctx, data.Id(), groupssMembershipType, groups, m)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = updateRepos(ctx, data.Id(), repos, m)
		if err != nil {
			return diag.FromErr(err)
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

		_, err = m.(*resty.Client).R().
			SetPathParam("projectKey", data.Id()).
			SetBody(project).
			Put(projectUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(project.Id())

		// Role should be updated first before members or groups as they may depend on roles defined by the users
		_, err = updateRoles(ctx, data.Id(), roles, m)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = updateMembers(ctx, data.Id(), usersMembershipType, users, m)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = updateMembers(ctx, data.Id(), groupssMembershipType, groups, m)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = updateRepos(ctx, data.Id(), repos, m)
		if err != nil {
			return diag.FromErr(err)
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

		g := new(errgroup.Group)
		deleteRepos(ctx, data.Id(), repos, m, g)
		if err := g.Wait(); err != nil {
			return diag.FromErr(err)
		}

		resp, err := m.(*resty.Client).R().
			SetPathParam("projectKey", data.Id()).
			Delete(projectUrl)

		if err != nil && resp.StatusCode() == http.StatusNotFound {
			data.SetId("")
			return diag.FromErr(err)
		}

		return nil
	}

	return &schema.Resource{
		SchemaVersion: 1,
		CreateContext: createProject,
		ReadContext:   readProject,
		UpdateContext: updateProject,
		DeleteContext: deleteProject,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema:      projectSchema,
		Description: "Provides an Artifactory project resource. This can be used to create and manage Artifactory project, maintain users/groups/roles/repos.",
	}
}
