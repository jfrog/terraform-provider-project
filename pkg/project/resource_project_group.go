package project

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/jfrog/terraform-provider-shared/util/sdk"
	"github.com/jfrog/terraform-provider-shared/validator"
)

const projectGroupsUrl = "access/api/v1/projects/{projectKey}/groups/{name}"

type ProjectGroup struct {
	ProjectKey string   `json:"-"`
	Name       string   `json:"name"`
	Roles      []string `json:"roles"`
}

func (m ProjectGroup) Id() string {
	return fmt.Sprintf(`%s:%s`, m.ProjectKey, m.Name)
}

func projectGroupResource() *schema.Resource {
	var projectGroupSchema = map[string]*schema.Schema{
		"project_key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.ProjectKey,
			Description:      "The key of the project to which the group should be assigned to.",
		},
		"name": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotEmpty),
			Description:      "The name of an artifactory group.",
		},
		"roles": {
			Type:        schema.TypeSet,
			Required:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "List of pre-defined Project or custom roles",
		},
	}

	var unpackProjectGroup = func(d *schema.ResourceData) ProjectGroup {
		return ProjectGroup{
			ProjectKey: d.Get("project_key").(string),
			Name:       d.Get("name").(string),
			Roles:      sdk.CastToStringArr(d.Get("roles").(*schema.Set).List()),
		}
	}

	var packProjectGroup = func(_ context.Context, data *schema.ResourceData, m ProjectGroup) diag.Diagnostics {
		setValue := sdk.MkLens(data)

		setValue("name", m.Name)
		setValue("project_key", m.ProjectKey)
		errors := setValue("roles", m.Roles)

		if len(errors) > 0 {
			return diag.Errorf("failed to pack project member %q", errors)
		}

		return nil
	}

	var readProjectGroup = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectGroup := unpackProjectGroup(data)
		var loadedProjectGroup ProjectGroup

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectGroup.ProjectKey,
				"name":       projectGroup.Name,
			}).
			SetResult(&loadedProjectGroup).
			Get(projectGroupsUrl)

		if err != nil {
			return diag.FromErr(err)
		}

		loadedProjectGroup.ProjectKey = projectGroup.ProjectKey

		return packProjectGroup(ctx, data, loadedProjectGroup)
	}

	var upsertProjectGroup = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectGroup := unpackProjectGroup(data)

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectGroup.ProjectKey,
				"name":       projectGroup.Name,
			}).
			SetBody(&projectGroup).
			Put(projectGroupsUrl)

		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(projectGroup.Id())

		return readProjectGroup(ctx, data, m)
	}

	var deleteProjectGroup = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectGroup := unpackProjectGroup(data)

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectGroup.ProjectKey,
				"name":       projectGroup.Name,
			}).
			Delete(projectGroupsUrl)

		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId("")

		return nil
	}

	var importForProjectKeyGroupName = func(d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
		parts := strings.SplitN(d.Id(), ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("unexpected format of ID (%s), expected project_key:name", d.Id())
		}

		d.Set("project_key", parts[0])
		d.Set("name", parts[1])

		return []*schema.ResourceData{d}, nil
	}

	return &schema.Resource{
		CreateContext: upsertProjectGroup,
		ReadContext:   readProjectGroup,
		UpdateContext: upsertProjectGroup,
		DeleteContext: deleteProjectGroup,

		Importer: &schema.ResourceImporter{
			State: importForProjectKeyGroupName,
		},

		Schema:        projectGroupSchema,
		SchemaVersion: 1,

		Description: "Add a group as project member. Element has one to one mapping with the [JFrog Project Groups API](https://jfrog.com/help/r/jfrog-rest-apis/update-group-in-project). Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.",
	}
}
