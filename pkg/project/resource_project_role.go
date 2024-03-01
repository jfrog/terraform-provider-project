package project

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/jfrog/terraform-provider-shared/validator"
)

const projectRolesUrl = projectUrl + "/roles"
const projectRoleUrl = projectRolesUrl + "/{roleName}"

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

type Role struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Environments []string `json:"environments"`
	Actions      []string `json:"actions"`
}

func (r Role) Id() string {
	return r.Name
}

func (a Role) Equals(b Equatable) bool {
	return a.Id() == b.Id()
}

func projectRoleResource() *schema.Resource {
	var projectRoleSchema = map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.All(
				validation.StringIsNotEmpty,
				maxLength(64),
			)),
		},
		"type": {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(customRoleTypeRegex, fmt.Sprintf(`Only "%s" is supported`, customRoleType))),
			Description:      fmt.Sprintf(`Type of role. Only "%s" is supported`, customRoleType),
		},
		"project_key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.ProjectKey,
			Description:      "Project key for this environment. This field supports only 2 - 20 lowercase alphanumeric and hyphen characters. Must begin with a letter.",
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
	}

	var packRole = func(_ context.Context, data *schema.ResourceData, role Role, projectKey string) diag.Diagnostics {
		setValue := util.MkLens(data)

		setValue("name", role.Name)
		setValue("type", role.Type)
		setValue("project_key", projectKey)
		setValue("environments", role.Environments)
		errors := setValue("actions", role.Actions)

		if len(errors) > 0 {
			return diag.Errorf("failed to pack project role %q", errors)
		}

		return nil
	}

	var readProjectRole = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		var role Role
		projectKey := data.Get("project_key").(string)

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectKey,
				"roleName":   data.Id(),
			}).
			SetResult(&role).
			Get(projectRoleUrl)

		if err != nil {
			return diag.FromErr(err)
		}

		return packRole(ctx, data, role, projectKey)
	}

	var unpackRole = func(data *schema.ResourceData) Role {
		d := &util.ResourceData{ResourceData: data}

		return Role{
			Name:         d.GetString("name", false),
			Type:         d.GetString("type", false),
			Environments: d.GetSet("environments"),
			Actions:      d.GetSet("actions"),
		}
	}

	var createProjectRole = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectKey := data.Get("project_key").(string)
		role := unpackRole(data)

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParam("projectKey", projectKey).
			SetBody(role).
			Post(projectRolesUrl)

		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(role.Id())

		return readProjectRole(ctx, data, m)
	}

	var updateProjectRole = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectKey := data.Get("project_key").(string)
		role := unpackRole(data)

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectKey,
				"roleName":   role.Name,
			}).
			SetBody(role).
			Put(projectRoleUrl)

		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(role.Id())

		return readProjectRole(ctx, data, m)
	}

	var deleteProjectRole = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		resp, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"roleName":   data.Id(),
				"projectKey": data.Get("project_key").(string),
			}).
			Delete(projectRoleUrl)

		if err != nil {
			if resp.StatusCode() == http.StatusNotFound {
				data.SetId("")
			}
			return diag.FromErr(err)
		}

		return nil
	}

	var importForProjectKeyRoleName = func(d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
		parts := strings.SplitN(d.Id(), ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("unexpected format of ID (%s), expected project_key:role_name", d.Id())
		}

		d.Set("project_key", parts[0])
		d.Set("name", parts[1])
		d.SetId(parts[1])

		return []*schema.ResourceData{d}, nil
	}

	return &schema.Resource{
		SchemaVersion: 1,
		CreateContext: createProjectRole,
		ReadContext:   readProjectRole,
		UpdateContext: updateProjectRole,
		DeleteContext: deleteProjectRole,

		Importer: &schema.ResourceImporter{
			State: importForProjectKeyRoleName,
		},

		Schema:      projectRoleSchema,
		Description: "Create a project role. Element has one to one mapping with the [JFrog Project Roles API](https://www.jfrog.com/confluence/display/JFROG/Artifactory+REST+API#ArtifactoryRESTAPI-AddaNewRole). Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.",
	}
}
