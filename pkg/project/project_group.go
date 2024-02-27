package project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jfrog/terraform-provider-shared/util"
)

type ProjectGroup struct {
	ProjectKey string   `json:"-"`
	Name       string   `json:"name"`
	Roles      []string `json:"roles"`
}

func unpackProjectGroup(d *schema.ResourceData) ProjectGroup {
	return ProjectGroup{
		ProjectKey: d.Get("project_key").(string),
		Name:       d.Get("name").(string),
		Roles:      util.CastToStringArr(d.Get("roles").(*schema.Set).List()),
	}
}

func packProjectGroup(ctx context.Context, data *schema.ResourceData, m ProjectGroup) diag.Diagnostics {
	setValue := util.MkLens(data)

	errors := []error{}
	errors = append(errors, setValue("name", m.Name)...)
	errors = append(errors, setValue("project_key", m.ProjectKey)...)
	errors = append(errors, setValue("roles", m.Roles)...)

	if len(errors) > 0 {
		return diag.Errorf("failed to pack project member %q", errors)
	}

	return nil
}

func (m ProjectGroup) Id() string {
	return fmt.Sprintf(`%s:%s`, m.ProjectKey, m.Name)
}
