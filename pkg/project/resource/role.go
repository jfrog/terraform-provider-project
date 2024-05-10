package project

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func filterRoles(roles []Role, roleType string) []Role {
	filteredRoles := roles[:0]
	for _, role := range roles {
		if role.Type == roleType {
			filteredRoles = append(filteredRoles, role)
		}
	}

	return filteredRoles
}

var readRoles = func(ctx context.Context, projectKey string, client *resty.Client) ([]Role, error) {
	tflog.Debug(ctx, "readRoles")

	var roles []Role

	var projectError ProjectErrorsResponse
	resp, err := client.R().
		SetPathParam("projectKey", projectKey).
		SetResult(&roles).
		SetError(&projectError).
		Get(ProjectRolesUrl)

	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("%s", projectError.String())
	}

	tflog.Trace(ctx, fmt.Sprintf("roles: %+v\n", roles))

	// REST API returns all project roles, including ones with PREDEFINED type which can't be altered.
	// We are only interested in the "CUSTOM" types that we can manipulate.
	customRoles := filterRoles(roles, customRoleType)
	tflog.Trace(ctx, fmt.Sprintf("customRoles: %+v\n", customRoles))

	return customRoles, nil
}

var updateRoles = func(ctx context.Context, projectKey string, terraformRoles []Role, client *resty.Client) ([]Role, error) {
	tflog.Debug(ctx, "updateRoles")
	tflog.Trace(ctx, fmt.Sprintf("terraformRoles: %+v\n", terraformRoles))

	projectRoles, err := readRoles(ctx, projectKey, client)
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
		err := addRole(ctx, projectKey, role, client)
		if err != nil {
			return nil, fmt.Errorf("failed to add role %s: %s", role, err)
		}
	}

	for _, role := range rolesToBeUpdated {
		err := updateRole(ctx, projectKey, role, client)
		if err != nil {
			return nil, fmt.Errorf("failed to update role %s: %s", role, err)
		}
	}

	deleteErr := deleteRoles(ctx, projectKey, rolesToBeDeleted, client)
	if deleteErr != nil {
		return nil, fmt.Errorf("failed to delete roles for project: %s", deleteErr)
	}

	return readRoles(ctx, projectKey, client)
}

var addRole = func(ctx context.Context, projectKey string, role Role, client *resty.Client) error {
	tflog.Debug(ctx, "addRole")

	var projectError ProjectErrorsResponse
	resp, err := client.R().
		SetPathParam("projectKey", projectKey).
		SetBody(role).
		SetError(&projectError).
		Post(ProjectRolesUrl)
	if err != nil {
		tflog.Debug(ctx, "addRole", map[string]interface{}{
			"err": err,
		})
		return err
	}
	if resp.IsError() {
		tflog.Debug(ctx, "addRole", map[string]interface{}{
			"projectError": projectError,
		})
		return fmt.Errorf("%s", projectError.String())
	}

	return nil
}

var updateRole = func(ctx context.Context, projectKey string, role Role, client *resty.Client) error {
	tflog.Debug(ctx, "updateRole")

	var projectError ProjectErrorsResponse
	resp, err := client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   role.Name,
		}).
		SetBody(role).
		SetError(&projectError).
		Put(ProjectRoleUrl)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", projectError.String())
	}

	return nil
}

var deleteRoles = func(ctx context.Context, projectKey string, roles []Role, client *resty.Client) error {
	tflog.Debug(ctx, "deleteRoles")

	for _, role := range roles {
		err := deleteRole(ctx, projectKey, role, client)
		if err != nil {
			return fmt.Errorf("failed to delete role %s: %s", role, err)
		}
	}

	return nil
}

var deleteRole = func(ctx context.Context, projectKey string, role Role, client *resty.Client) error {
	tflog.Debug(ctx, "deleteRole")
	tflog.Trace(ctx, fmt.Sprintf("%+v\n", role))

	var projectError ProjectErrorsResponse
	resp, err := client.R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   role.Name,
		}).
		SetBody(role).
		SetError(&projectError).
		Delete(ProjectRoleUrl)

	if err != nil {
		return err
	}
	if resp.IsError() && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("%s", projectError.String())
	}

	return nil
}
