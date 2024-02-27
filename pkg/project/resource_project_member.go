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

const projectMembersUrl = "access/api/v1/projects/{projectKey}/users/{name}"

func projectMemberResource() *schema.Resource {
	var projectMemberSchema = map[string]*schema.Schema{
		"project_key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.ProjectKey,
			Description:      "The key of the project to which the member belongs.",
		},
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
		"ignore_missing_user": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "When set to true, the resource will not fail if the user does not exist. Default to false. This is useful when the user is externally managed and the local account wasn't created yet.",
		},
	}

	var readProjectMember = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectMember := unpackProjectMember(data)
		var loadedProjectMember ProjectMember

		resp, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectMember.ProjectKey,
				"name":       projectMember.Name,
			}).
			SetResult(&loadedProjectMember).
			Get(projectMembersUrl)

		if resp != nil && resp.StatusCode() == http.StatusNotFound && projectMember.IgnoreMissingUser {
			// ignore missing user, reuse local info for state
			loadedProjectMember = projectMember
		} else if err != nil {
			return diag.FromErr(err)
		}

		loadedProjectMember.ProjectKey = projectMember.ProjectKey
		loadedProjectMember.IgnoreMissingUser = projectMember.IgnoreMissingUser

		return packProjectMember(ctx, data, loadedProjectMember)
	}

	var upsertProjectMember = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectMember := unpackProjectMember(data)

		resp, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectMember.ProjectKey,
				"name":       projectMember.Name,
			}).
			SetBody(&projectMember).
			Put(projectMembersUrl)

		// allow missing user? -> report warning and ignore error
		diagnostics := diag.Diagnostics{}

		if resp != nil && resp.StatusCode() == http.StatusNotFound {
			if projectMember.IgnoreMissingUser {
				diagnostics = append(diagnostics, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  fmt.Sprintf("user '%s' not found, but ignore_missing_user is set to true, project membership not created", projectMember.Name),
				})
			} else {
				return diag.Errorf("user '%s' not found, project membership not created", projectMember.Name)
			}
		} else if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(projectMember.Id())

		diagnostics = append(diagnostics, readProjectMember(ctx, data, m)...)

		if len(diagnostics) > 0 {
			return diagnostics
		}

		return nil
	}

	var deleteProjectMember = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectMember := unpackProjectMember(data)

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectMember.ProjectKey,
				"name":       projectMember.Name,
			}).
			Delete(projectMembersUrl)

		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId("")

		return nil
	}

	var importForProjectKeyMemberName = func(d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
		parts := strings.SplitN(d.Id(), ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("unexpected format of ID (%s), expected project_key:name", d.Id())
		}

		d.Set("project_key", parts[0])
		d.Set("name", parts[1])

		return []*schema.ResourceData{d}, nil
	}

	return &schema.Resource{
		CreateContext: upsertProjectMember,
		ReadContext:   readProjectMember,
		UpdateContext: upsertProjectMember,
		DeleteContext: deleteProjectMember,

		Importer: &schema.ResourceImporter{
			State: importForProjectKeyMemberName,
		},

		Schema:        projectMemberSchema,
		SchemaVersion: 1,

		Description: "Create a project member. Element has one to one mapping with the [JFrog Project Users API](https://jfrog.com/help/r/jfrog-rest-apis/add-or-update-user-in-project). Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.",
	}
}
