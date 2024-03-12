package project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/jfrog/terraform-provider-shared/util/sdk"
)

var unpackRoles = func(data *schema.ResourceData) []Role {
	var roles []Role

	if v, ok := data.GetOk("role"); ok {
		projectRoles := v.(*schema.Set).List()
		if len(projectRoles) == 0 {
			return roles
		}

		for _, projectRole := range projectRoles {
			id := projectRole.(map[string]interface{})

			role := Role{
				Name:         id["name"].(string),
				Description:  id["description"].(string),
				Type:         id["type"].(string),
				Environments: sdk.CastToStringArr(id["environments"].(*schema.Set).List()),
				Actions:      sdk.CastToStringArr(id["actions"].(*schema.Set).List()),
			}
			roles = append(roles, role)
		}
	}

	return roles
}

var packRoles = func(ctx context.Context, d *schema.ResourceData, roles []Role) []error {
	tflog.Debug(ctx, "packRoles")

	setValue := sdk.MkLens(d)

	var projectRoles []interface{}

	for _, role := range roles {
		tflog.Trace(ctx, fmt.Sprintf("%+v\n", role))
		projectRole := map[string]interface{}{
			"name":         role.Name,
			"description":  role.Description,
			"type":         role.Type,
			"environments": role.Environments,
			"actions":      role.Actions,
		}

		projectRoles = append(projectRoles, projectRole)
	}

	tflog.Trace(ctx, fmt.Sprintf("%+v\n", projectRoles))

	errors := setValue("role", projectRoles)

	return errors
}

func filterRoles(roles []Role, roleType string) []Role {
	filteredRoles := roles[:0]
	for _, role := range roles {
		if role.Type == roleType {
			filteredRoles = append(filteredRoles, role)
		}
	}

	return filteredRoles
}

var readRoles = func(ctx context.Context, projectKey string, m interface{}) ([]Role, error) {
	tflog.Debug(ctx, "readRoles")

	roles := []Role{}

	_, err := m.(util.ProvderMetadata).Client.R().
		SetPathParam("projectKey", projectKey).
		SetResult(&roles).
		Get(projectRolesUrl)

	if err != nil {
		return nil, err
	}

	tflog.Trace(ctx, fmt.Sprintf("roles: %+v\n", roles))

	// REST API returns all project roles, including ones with PREDEFINED type which can't be altered.
	// We are only interested in the "CUSTOM" types that we can manipulate.
	customRoles := filterRoles(roles, customRoleType)
	tflog.Trace(ctx, fmt.Sprintf("customRoles: %+v\n", customRoles))

	return customRoles, nil
}

var updateRoles = func(ctx context.Context, projectKey string, terraformRoles []Role, m interface{}) ([]Role, error) {
	tflog.Debug(ctx, "updateRoles")
	tflog.Trace(ctx, fmt.Sprintf("terraformRoles: %+v\n", terraformRoles))

	projectRoles, err := readRoles(ctx, projectKey, m)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles for project: %s", err)
	}
	tflog.Trace(ctx, fmt.Sprintf("projectRoles: %+v\n", projectRoles))

	terraformRolesSet := SetFromSlice(terraformRoles)
	projectRolesSet := SetFromSlice(projectRoles)

	rolesToBeAdded := terraformRolesSet.Difference(projectRolesSet)
	tflog.Trace(ctx, fmt.Sprintf("rolesToBeAdded: %+v\n", rolesToBeAdded))

	rolesToBeUpdated := terraformRolesSet.Intersection(projectRolesSet)
	tflog.Trace(ctx, fmt.Sprintf("rolesToBeUpdated: %+v\n", rolesToBeUpdated))

	rolesToBeDeleted := projectRolesSet.Difference(terraformRolesSet)
	tflog.Trace(ctx, fmt.Sprintf("rolesToBeDeleted: %+v\n", rolesToBeDeleted))

	for _, role := range rolesToBeAdded {
		err := addRole(ctx, projectKey, role, m)
		if err != nil {
			return nil, fmt.Errorf("failed to add role %s: %s", role, err)
		}
	}

	for _, role := range rolesToBeUpdated {
		err := updateRole(ctx, projectKey, role, m)
		if err != nil {
			return nil, fmt.Errorf("failed to update role %s: %s", role, err)
		}
	}

	deleteErr := deleteRoles(ctx, projectKey, rolesToBeDeleted, m)
	if deleteErr != nil {
		return nil, fmt.Errorf("failed to delete roles for project: %s", deleteErr)
	}

	return readRoles(ctx, projectKey, m)
}

var addRole = func(ctx context.Context, projectKey string, role Role, m interface{}) error {
	tflog.Debug(ctx, "addRole")

	_, err := m.(util.ProvderMetadata).Client.R().
		SetPathParam("projectKey", projectKey).
		SetBody(role).
		Post(projectRolesUrl)

	return err
}

var updateRole = func(ctx context.Context, projectKey string, role Role, m interface{}) error {
	tflog.Debug(ctx, "updateRole")

	_, err := m.(util.ProvderMetadata).Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   role.Name,
		}).
		SetBody(role).
		Put(projectRoleUrl)

	return err
}

var deleteRoles = func(ctx context.Context, projectKey string, roles []Role, m interface{}) error {
	tflog.Debug(ctx, "deleteRoles")

	for _, role := range roles {
		err := deleteRole(ctx, projectKey, role, m)
		if err != nil {
			return fmt.Errorf("failed to delete role %s: %s", role, err)
		}
	}

	return nil
}

var deleteRole = func(ctx context.Context, projectKey string, role Role, m interface{}) error {
	tflog.Debug(ctx, "deleteRole")
	tflog.Trace(ctx, fmt.Sprintf("%+v\n", role))

	_, err := m.(util.ProvderMetadata).Client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   role.Name,
		}).
		SetBody(role).
		Delete(projectRoleUrl)

	if err != nil {
		return err
	}

	return nil
}
