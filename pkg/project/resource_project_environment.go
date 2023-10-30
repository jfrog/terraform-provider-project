package project

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/jfrog/terraform-provider-shared/validator"
)

type ProjectEnvironment struct {
	Name string `json:"name"`
}

func (p ProjectEnvironment) Id() string {
	return p.Name
}

type ProjectEnvironmentUpdate struct {
	NewName string `json:"new_name"`
}

func (p ProjectEnvironmentUpdate) Id() string {
	return p.NewName
}

const projectEnvironmentUrl = "/access/api/v1/projects/{projectKey}/environments"

func projectEnvironmentResource() *schema.Resource {

	var projectEnvironmentSchema = map[string]*schema.Schema{
		"name": {
			Required: true,
			Type:     schema.TypeString,
			ValidateDiagFunc: validation.ToDiagFunc(validation.All(
				validation.StringIsNotEmpty,
				validation.StringMatch(regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]+$`), "Must start with a letter and contain letters, digits and `-` character."),
			)),
			Description: "Environment name. Must start with a letter and can contain letters, digits and `-` character.",
		},
		"project_key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.ProjectKey,
			Description:      "Project key for this environment. This field supports only 2 - 20 lowercase alphanumeric and hyphen characters. Must begin with a letter.",
		},
	}

	var readProjectEnvironment = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectKey := data.Get("project_key").(string)
		var envs []ProjectEnvironment

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParam("projectKey", projectKey).
			SetResult(&envs).
			Get(projectEnvironmentUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		var matchedEnv *ProjectEnvironment
		for _, env := range envs {
			if env.Name == fmt.Sprintf("%s-%s", projectKey, data.Get("name")) {
				matchedEnv = &env
				break
			}
		}

		if matchedEnv == nil {
			data.SetId("")
			return nil
		}

		data.Set("name", strings.TrimPrefix(matchedEnv.Name, fmt.Sprintf("%s-", projectKey)))
		data.Set("project_key", projectKey)
		data.SetId(matchedEnv.Id())

		return nil
	}

	var createProjectEnvironment = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectKey := data.Get("project_key").(string)
		projectEnvironment := ProjectEnvironment{
			Name: fmt.Sprintf("%s-%s", projectKey, data.Get("name").(string)),
		}

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParam("projectKey", projectKey).
			SetBody(projectEnvironment).
			Post(projectEnvironmentUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(projectEnvironment.Id())

		return readProjectEnvironment(ctx, data, m)
	}

	var updateProjectEnvironment = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectKey := data.Get("project_key").(string)
		oldName, newName := data.GetChange("name")

		projectEnvironmentUpdate := ProjectEnvironmentUpdate{
			NewName: fmt.Sprintf("%s-%s", projectKey, newName),
		}

		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey":      projectKey,
				"environmentName": fmt.Sprintf("%s-%s", projectKey, oldName),
			}).
			SetBody(projectEnvironmentUpdate).
			Post(projectEnvironmentUrl + "/{environmentName}/rename")
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(projectEnvironmentUpdate.Id())
		data.Set("name", newName)

		return readProjectEnvironment(ctx, data, m)
	}

	var deleteProjectEnvironment = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectKey := data.Get("project_key").(string)
		_, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey":      projectKey,
				"environmentName": fmt.Sprintf("%s-%s", projectKey, data.Get("name")),
			}).
			Delete(projectEnvironmentUrl + "/{environmentName}")
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId("")

		return nil
	}

	var importForProjectKeyEnvironmentName = func(d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
		parts := strings.SplitN(d.Id(), ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("unexpected format of ID (%s), expected project_key:environment_name", d.Id())
		}

		d.Set("project_key", parts[0])
		d.Set("name", parts[1])
		d.SetId(fmt.Sprintf("%s-%s", parts[0], parts[1]))

		return []*schema.ResourceData{d}, nil
	}

	var projectEnvironmentLengthDiff = func(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
		projectEnvironmentName := fmt.Sprintf("%s-%s", diff.Get("project_key"), diff.Get("name"))
		tflog.Debug(ctx, fmt.Sprintf("projectEnvironmentName: %s", projectEnvironmentName))

		if len(projectEnvironmentName) > 32 {
			return fmt.Errorf("combined length of project_key and name (separated by '-') cannot exceed 32 characters")
		}

		return nil
	}

	return &schema.Resource{
		SchemaVersion: 1,
		CreateContext: createProjectEnvironment,
		ReadContext:   readProjectEnvironment,
		UpdateContext: updateProjectEnvironment,
		DeleteContext: deleteProjectEnvironment,

		Importer: &schema.ResourceImporter{
			State: importForProjectKeyEnvironmentName,
		},

		CustomizeDiff: projectEnvironmentLengthDiff,

		Schema:      projectEnvironmentSchema,
		Description: "Creates a new environment for the specified project.\n\n~>The combined length of `project_key` and `name` (separated by '-') cannot not exceeds 32 characters.",
	}
}
