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
	"github.com/jfrog/terraform-provider-shared/util/sdk"
	"github.com/jfrog/terraform-provider-shared/validator"
)

const projectUsersUrl = "access/api/v1/projects/{projectKey}/users/{name}"

type ProjectUser struct {
	ProjectKey        string   `json:"-"`
	Name              string   `json:"name"`
	Roles             []string `json:"roles"`
	IgnoreMissingUser bool     `json:"-"`
}

func (m ProjectUser) Id() string {
	return fmt.Sprintf(`%s:%s`, m.ProjectKey, m.Name)
}

func projectUserResource() *schema.Resource {
	var projectUserSchema = map[string]*schema.Schema{
		"project_key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.ProjectKey,
			Description:      "The key of the project to which the user should be assigned to.",
		},
		"name": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
			Description:      "The name of an artifactory user.",
		},
		"roles": {
			Type:        schema.TypeSet,
			Required:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "List of pre-defined Project or custom roles",
		},
		"ignore_missing_user": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "When set to `true`, the resource will not fail if the user does not exist. Default to `false`. This is useful when the user is externally managed and the local account wasn't created yet.",
		},
	}

	var packProjectUser = func(_ context.Context, data *schema.ResourceData, m ProjectUser) diag.Diagnostics {
		setValue := sdk.MkLens(data)

		setValue("name", m.Name)
		setValue("project_key", m.ProjectKey)
		setValue("roles", m.Roles)
		errors := setValue("ignore_missing_user", m.IgnoreMissingUser)

		if len(errors) > 0 {
			return diag.Errorf("failed to pack project member %q", errors)
		}

		return nil
	}

	var unpackProjectUser = func(d *schema.ResourceData) ProjectUser {
		return ProjectUser{
			ProjectKey:        d.Get("project_key").(string),
			Name:              d.Get("name").(string),
			Roles:             sdk.CastToStringArr(d.Get("roles").(*schema.Set).List()),
			IgnoreMissingUser: d.Get("ignore_missing_user").(bool),
		}
	}

	var readProjectUser = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectUser := unpackProjectUser(data)
		var loadedProjectUser ProjectUser

		var projectError ProjectErrorsResponse
		resp, err := m.(util.ProviderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectUser.ProjectKey,
				"name":       projectUser.Name,
			}).
			SetError(&projectError).
			SetResult(&loadedProjectUser).
			Get(projectUsersUrl)

		if err != nil {
			return diag.FromErr(err)
		}
		if resp.StatusCode() == http.StatusNotFound && projectUser.IgnoreMissingUser {
			// ignore missing user, reuse local info for state
			loadedProjectUser = projectUser
		} else if resp.IsError() {
			return diag.Errorf("%s", projectError.String())
		}

		loadedProjectUser.ProjectKey = projectUser.ProjectKey
		loadedProjectUser.IgnoreMissingUser = projectUser.IgnoreMissingUser

		return packProjectUser(ctx, data, loadedProjectUser)
	}

	var upsertProjectUser = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectUser := unpackProjectUser(data)

		var projectError ProjectErrorsResponse
		resp, err := m.(util.ProviderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectUser.ProjectKey,
				"name":       projectUser.Name,
			}).
			SetBody(&projectUser).
			SetError(&projectError).
			Put(projectUsersUrl)

		// allow missing user? -> report warning and ignore error
		diagnostics := diag.Diagnostics{}

		if err != nil {
			return diag.FromErr(err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			if projectUser.IgnoreMissingUser {
				diagnostics = append(diagnostics, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("user '%s' not found, but ignore_missing_user is set to true, project membership not created", projectUser.Name),
				})
			} else {
				return diag.Errorf("user '%s' not found, project membership not created", projectUser.Name)
			}
		} else if resp.IsError() {
			return diag.Errorf("%s", projectError.String())
		}

		data.SetId(projectUser.Id())

		diagnostics = append(diagnostics, readProjectUser(ctx, data, m)...)

		if len(diagnostics) > 0 {
			return diagnostics
		}

		return nil
	}

	var deleteProjectUser = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectUser := unpackProjectUser(data)

		var projectError ProjectErrorsResponse
		resp, err := m.(util.ProviderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectUser.ProjectKey,
				"name":       projectUser.Name,
			}).
			SetError(&projectError).
			Delete(projectUsersUrl)

		if err != nil {
			return diag.FromErr(err)
		}
		if resp.IsError() && resp.StatusCode() != http.StatusNotFound {
			return diag.Errorf("%s", projectError.String())
		}

		data.SetId("")

		return nil
	}

	var importForProjectKeyUserName = func(d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
		parts := strings.SplitN(d.Id(), ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("unexpected format of ID (%s), expected project_key:name", d.Id())
		}

		d.Set("project_key", parts[0])
		d.Set("name", parts[1])

		return []*schema.ResourceData{d}, nil
	}

	return &schema.Resource{
		CreateContext: upsertProjectUser,
		ReadContext:   readProjectUser,
		UpdateContext: upsertProjectUser,
		DeleteContext: deleteProjectUser,

		Importer: &schema.ResourceImporter{
			State: importForProjectKeyUserName,
		},

		Schema:        projectUserSchema,
		SchemaVersion: 1,

		Description: "Add a user as project member. Element has one to one mapping with the [JFrog Project Users API](https://jfrog.com/help/r/jfrog-rest-apis/add-or-update-user-in-project). Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.",
	}
}
