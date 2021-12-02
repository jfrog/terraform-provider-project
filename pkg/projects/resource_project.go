package projects

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strconv"
)

type Identifiable interface {
	Id() string
}

type AdminPrivileges struct {
	ManageMembers   bool `json:"manage_members"`
	ManageResources bool `json:"manage_resources"`
	IndexResources  bool `json:"index_resources"`
}

// Project GET {{ host }}/access/api/v1/projects/{{prjKey}}/
//GET {{ host }}/artifactory/api/repositories/?prjKey={{prjKey}}
type Project struct {
	Key                    string           `json:"project_key"`
	DisplayName            string           `json:"display_name"`
	Description            string           `json:"description"`
	AdminPrivileges        AdminPrivileges  `json:"admin_privileges"`
	StorageQuota           int              `json:"storage_quota_bytes"`
	SoftLimit              bool             `json:"soft_limit"`
	QuotaEmailNotification bool             `json:"storage_quota_email_notification"`
}

func (p Project) Id() string {
	return p.Key
}

const projectsUrl = "/access/api/v1/projects/"
const projectUsersUrl = projectsUrl + "%s/users/"
const projectGroupsUrl = projectsUrl + "%s/groups/"

func verifyProject(id string, request *resty.Request) (*resty.Response, error) {
	return request.Head(projectsUrl + id)
}

func projectResource() *schema.Resource {

	var projectSchema = map[string]*schema.Schema{
		"key": {
			Type:     schema.TypeString,
			Required: true,
			ValidateDiagFunc: validation.ToDiagFunc(
				validation.StringMatch(regexp.MustCompile("^[a-z0-9]{3,6}$"), "key must be 3 - 6 lowercase alphanumeric characters"),
			),
			Description: "The Project Key is added as a prefix to resources created within a Project. This field is mandatory and supports only 3 - 6 lowercase alphanumeric characters. Must begin with a letter. For example: us1a.",
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
			Description: "Storage quota in GB. Must be 1 or larger. Set to -1 for unlimited storage.",
		},
		"block_deployments_on_limit": {
			Type:       schema.TypeBool,
			Optional:   true,
			Default:    false,
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
	}

	var unpackProject = func(data *schema.ResourceData) (string, Project, Membership, Membership, error) {
		d := &ResourceData{data}

		project := Project{
			Key:                    d.getString("key", false),
			DisplayName:            d.getString("display_name", false),
			Description:            d.getString("description", false),
			StorageQuota:           GibibytesToBytes(d.getInt("max_storage_in_gibibytes", false)),
			SoftLimit:              d.getBool("block_deployments_on_limit", false),
			QuotaEmailNotification: d.getBool("email_notification", false),
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

		_, users, err := unpackMembers(data, "member")
		_, groups, err := unpackMembers(data, "group")

		return project.Id(), project, users, groups, err
	}

	var packProject = func(d *schema.ResourceData, project *Project, users []Member, groups []Member) diag.Diagnostics {
		var errors []error
		setValue := mkLens(d)

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
			errors = packMembers(d, "member", users)
		}

		if len(groups) > 0 {
			errors = packMembers(d, "group", groups)
		}

		if len(errors) > 0 {
			return diag.Errorf("failed to pack project %q", errors)
		}

		return nil
	}

	var readProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		project := Project{}

		_, err := m.(*resty.Client).R().SetResult(&project).Get(projectsUrl + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		users, err := readMembers(fmt.Sprintf(projectUsersUrl, data.Id()), m)
		if err != nil {
			return diag.FromErr(err)
		}

		groups, err := readMembers(fmt.Sprintf(projectGroupsUrl, data.Id()), m)
		if err != nil {
			return diag.FromErr(err)
		}

		return packProject(data, &project, users, groups)
	}

	var createProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] createProject")
		log.Printf("[TRACE] %+v\n", data)

		key, project, users, groups, err := unpackProject(data)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = m.(*resty.Client).R().SetBody(project).Post(projectsUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(key)

		_, err = updateMembers(fmt.Sprintf(projectUsersUrl, data.Id()), users, m)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = updateMembers(fmt.Sprintf(projectGroupsUrl, data.Id()), groups, m)
		if err != nil {
			return diag.FromErr(err)
		}

		return readProject(ctx, data, m)
	}

	var updateProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] updateProject")
		log.Printf("[TRACE] %+v\n", data)

		key, project, users, groups, err := unpackProject(data)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = m.(*resty.Client).R().SetBody(project).Put(projectsUrl + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(key)

		_, err = updateMembers(fmt.Sprintf(projectUsersUrl, data.Id()), users, m)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = updateMembers(fmt.Sprintf(projectGroupsUrl, data.Id()), groups, m)
		if err != nil {
			return diag.FromErr(err)
		}

		return readProject(ctx, data, m)
	}

	var deleteProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		log.Printf("[DEBUG] deleteProject")
		log.Printf("[TRACE] %+v\n", data)

		resp, err := m.(*resty.Client).R().Delete(projectsUrl + data.Id())
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

		Schema: projectSchema,
		Description: "Provides an Artifactory project resource. This can be used to create and manage Artifactory project, maintain users/groups/roles/repos.",
	}
}
