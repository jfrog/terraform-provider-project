package projects

import (
	"context"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strconv"
)
type Identifiable interface {
	Id() string
}
// Project GET {{ host }}/access/api/v1/projects/{{prjKey}}/
//GET {{ host }}/artifactory/api/repositories/?prjKey={{prjKey}}
type Project struct {
	DisplayName            string   `hcl:"display_name" json:"display_name,omitempty"`
	Description            string   `hcl:"description" json:"description,omitempty"`
	StorageQuota           int      `hcl:"max_storage" json:"storage_quota_bytes,omitempty"`
	SoftLimit              bool     `hcl:"block_deployments_on_limit" json:"soft_limit"`
	QuotaEmailNotification bool     `hcl:"email_notification" json:"storage_quota_email_notification"`
	Key                    string   `hcl:"key" json:"project_key"`
	Repositories           []string `hcl:"repositories" `
	Roles                  []Role   `hcl:"roles"`
	Users                  []Member `hcl:"users"`
	Groups                 []Group `hcl:"groups"`
}
func (p Project) Id() string {
	return p.Key
}

// Member GET {{ host }}/access/api/v1/projects/{{prjKey}}/users/
// GET {{ host }}/access/api/v1/projects/{{prjKey}}/groups/
type Member struct {
	Name  string   `hcl:"name" json:"name"`
	Roles []string `hcl:"roles" json:"roles"`
}
func (m Member) Id() string {
	return m.Name
}
type Group Member

func (g Group) Id() string {
	return g.Name
}



// Role GET {{ host }}/access/api/v1/projects/{{prjKey}}/roles/
// This gets all available project roles
type Role struct {
	Name         string   `hcl:"name" json:"name"`
	Description  string   `hcl:"description" json:"description"`
	Type         string   `hcl:"type" json:"type"`
	Environments []string `hcl:"environments" json:"environments"`
	Actions      []string `hcl:"actions" json:"actions"`
}

func (r Role) Id() string {
	return r.Name
}

func projectResource() *schema.Resource {

	const projectsUrl = "/access/api/v1/projects"
	var readProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		prj := Project{}
		_, err := m.(*resty.Client).R().SetResult(&prj).Post(projectsUrl + data.Id())

		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	var createProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		_, err := m.(*resty.Client).R().SetBody(nil).Post(projectsUrl)
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	}
	var updateProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		_, err := m.(*resty.Client).R().Put(projectsUrl + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	}
	var deleteProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		_, err := m.(*resty.Client).R().Delete(projectsUrl + data.Id())
		if err != nil {
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

		Schema: map[string]*schema.Schema{
			"key": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "The Project Key is added as a prefix to\nresources created within a Project.\nThis field is mandatory and supports only\n3 - 6 lowercase alphanumeric characters.\nMust begin with a letter.\nFor example: us1a.",
			},
			"display_name": {
				Required:         true,
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
				Description:      "Also known as project name on the UI",
			},
			"repositories": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
			"build_repository": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"quota": {
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
				Description: "Value in gigabytes. Alerts will be displayed when\nreaching 75% and 95% of the\nstorage quota.\nServes as a notification only and\nis not a blocker",
			},
			"block_deployments_on_limit": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"email_notification": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"groups": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"roles": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"members": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"roles": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"project_admins": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user_names": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"roles": {
							Type:     schema.TypeSet,
							Required: true,
							Set:      schema.HashString,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"admin_privs": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"manage_resources": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"index_resources": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"manage_members": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
		},
	}
}
