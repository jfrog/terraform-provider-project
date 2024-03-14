package project

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/jfrog/terraform-provider-shared/util/sdk"
	"github.com/jfrog/terraform-provider-shared/validator"
)

type Repository struct {
	Key        string `json:"key"`
	ProjectKey string `json:"projectKey"`
}

func projectRepositoryResource() *schema.Resource {
	var projectRepositorySchema = map[string]*schema.Schema{
		"project_key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.ProjectKey,
			Description:      "The key of the project to which the repository should be assigned to.",
		},
		"key": {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validator.RepoKey,
			Description:      "The key of the repository.",
		},
	}

	var readProjectRepository = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		repoKey := data.Get("key").(string)

		var repo Repository

		var projectError ProjectErrorsResponse
		resp, err := m.(util.ProvderMetadata).Client.R().
			SetResult(&repo).
			SetPathParam("key", repoKey).
			SetError(&projectError).
			Get("/artifactory/api/repositories/{key}")

		if err != nil {
			return diag.FromErr(err)
		}
		if resp.StatusCode() == http.StatusBadRequest || resp.StatusCode() == http.StatusNotFound {
			data.SetId("")
			return nil
		}
		if resp.IsError() {
			return diag.Errorf("%s", projectError.String())
		}

		if repo.ProjectKey == "" {
			tflog.Info(ctx, "no project_key for repo", map[string]any{"repoKey": repoKey})
			data.SetId("")
			return nil
		}

		setValue := sdk.MkLens(data)

		setValue("project_key", repo.ProjectKey)
		errors := setValue("key", repo.Key)

		if len(errors) > 0 {
			return diag.Errorf("failed to pack project repository %q", errors)
		}

		return nil
	}

	var createProjectRepository = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		projectKey := data.Get("project_key").(string)
		repoKey := data.Get("key").(string)

		var projectError ProjectErrorsResponse
		resp, err := m.(util.ProvderMetadata).Client.R().
			SetPathParams(map[string]string{
				"projectKey": projectKey,
				"repoKey":    repoKey,
			}).
			SetError(&projectError).
			Put("/access/api/v1/projects/_/attach/repositories/{repoKey}/{projectKey}?force=true")

		if err != nil {
			return diag.FromErr(err)
		}
		if resp.IsError() {
			return diag.Errorf("%s", projectError.String())
		}

		data.SetId(fmt.Sprintf("%s-%s", projectKey, repoKey))

		return readProjectRepository(ctx, data, m)
	}

	var deleteProjectRepository = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		repoKey := data.Get("key").(string)

		var projectError ProjectErrorsResponse
		resp, err := m.(util.ProvderMetadata).Client.R().
			SetPathParam("repoKey", repoKey).
			SetError(&projectError).
			Delete("/access/api/v1/projects/_/attach/repositories/{repoKey}")

		if err != nil {
			return diag.FromErr(err)
		}
		if resp.IsError() && resp.StatusCode() != http.StatusNotFound {
			return diag.Errorf("%s", projectError.String())
		}

		data.SetId("")

		return nil
	}

	var importForProjectKeyRepositoryKey = func(d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
		parts := strings.SplitN(d.Id(), ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("unexpected format of ID (%s), expected project_key:repository_key", d.Id())
		}

		d.Set("project_key", parts[0])
		d.Set("key", parts[1])
		d.SetId(fmt.Sprintf("%s-%s", parts[0], parts[1]))

		return []*schema.ResourceData{d}, nil
	}

	return &schema.Resource{
		CreateContext: createProjectRepository,
		ReadContext:   readProjectRepository,
		DeleteContext: deleteProjectRepository,

		Importer: &schema.ResourceImporter{
			State: importForProjectKeyRepositoryKey,
		},

		Schema:        projectRepositorySchema,
		SchemaVersion: 1,

		Description: "Assign a repository to a project. Requires a user assigned with the 'Administer the Platform' role or Project Admin permissions if `admin_privileges.manage_resoures` is enabled.",
	}
}
