package project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jfrog/terraform-provider-shared/util"
)

type ProjectUser struct {
	ProjectKey        string   `json:"-"`
	Name              string   `json:"name"`
	Roles             []string `json:"roles"`
	IgnoreMissingUser bool     `json:"-"`
}

func unpackProjectUser(d *schema.ResourceData) ProjectUser {
	return ProjectUser{
		ProjectKey:        d.Get("project_key").(string),
		Name:              d.Get("name").(string),
		Roles:             util.CastToStringArr(d.Get("roles").(*schema.Set).List()),
		IgnoreMissingUser: d.Get("ignore_missing_user").(bool),
	}
}

func packProjectUser(ctx context.Context, data *schema.ResourceData, m ProjectUser) diag.Diagnostics {
	setValue := util.MkLens(data)

	errors := []error{}
	errors = append(errors, setValue("name", m.Name)...)
	errors = append(errors, setValue("project_key", m.ProjectKey)...)
	errors = append(errors, setValue("roles", m.Roles)...)
	errors = append(errors, setValue("ignore_missing_user", m.IgnoreMissingUser)...)

	if len(errors) > 0 {
		return diag.Errorf("failed to pack project member %q", errors)
	}

	return nil
}

func (m ProjectUser) Id() string {
	return fmt.Sprintf(`%s:%s`, m.ProjectKey, m.Name)
}
