package project

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/sync/errgroup"
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

var unpackRoles = func(data *schema.ResourceData) []Role {
	d := &ResourceData{data}

	var roles []Role

	if v, ok := d.GetOkExists("role"); ok {
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
				Environments: castToStringArr(id["environments"].(*schema.Set).List()),
				Actions:      castToStringArr(id["actions"].(*schema.Set).List()),
			}
			roles = append(roles, role)
		}
	}

	return roles
}

var packRoles = func(d *schema.ResourceData, roles []Role) []error {
	log.Printf("[DEBUG] packRoles")

	setValue := mkLens(d)

	var projectRoles []interface{}

	for _, role := range roles {
		log.Printf("[TRACE] %+v\n", role)
		projectRole := map[string]interface{}{
			"name":         role.Name,
			"description":  role.Description,
			"type":         role.Type,
			"environments": role.Environments,
			"actions":      role.Actions,
		}

		projectRoles = append(projectRoles, projectRole)
	}

	log.Printf("[TRACE] %+v\n", projectRoles)

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

var readRoles = func(projectKey string, m interface{}) ([]Role, error) {
	log.Println("[DEBUG] readRoles")

	roles := []Role{}

	_, err := m.(*resty.Client).R().
		SetPathParam("projectKey", projectKey).
		SetResult(&roles).
		Get(projectRolesUrl)

	if err != nil {
		return nil, err
	}

	log.Printf("[TRACE] roles: %+v\n", roles)

	// REST API returns all project roles, including ones with PREDEFINED type which can't be altered.
	// We are only interested in the "CUSTOM" types that we can manipulate.
	customRoles := filterRoles(roles, customRoleType)
	log.Printf("[TRACE] customRoles: %+v\n", customRoles)

	return customRoles, nil
}

var updateRoles = func(projectKey string, terraformRoles []Role, m interface{}) ([]Role, error) {
	log.Println("[DEBUG] updateRoles")
	log.Printf("[TRACE] terraformRoles: %+v\n", terraformRoles)

	projectRoles, err := readRoles(projectKey, m)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roles for project: %s", err)
	}
	log.Printf("[TRACE] projectRoles: %+v\n", projectRoles)

	terraformRolesSet := SetFromSlice(terraformRoles)
	projectRolesSet := SetFromSlice(projectRoles)

	rolesToBeAdded := terraformRolesSet.Difference(projectRolesSet)
	log.Printf("[TRACE] rolesToBeAdded: %+v\n", rolesToBeAdded)

	rolesToBeUpdated := terraformRolesSet.Intersection(projectRolesSet)
	log.Printf("[TRACE] rolesToBeUpdated: %+v\n", rolesToBeUpdated)

	rolesToBeDeleted := projectRolesSet.Difference(terraformRolesSet)
	log.Printf("[TRACE] rolesToBeDeleted: %+v\n", rolesToBeDeleted)

	g := new(errgroup.Group)

	for _, role := range rolesToBeAdded {
		projectKey, role, m := projectKey, role, m

		g.Go(func() error {
			return addRole(projectKey, role, m)
		})
	}

	for _, role := range rolesToBeUpdated {
		projectKey, role, m := projectKey, role, m
		g.Go(func() error {
			return updateRole(projectKey, role, m)
		})
	}

	deleteRoles(projectKey, rolesToBeDeleted, m, g)

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to update roles for project: %s", err)
	}

	return readRoles(projectKey, m)
}

var addRole = func(projectKey string, role Role, m interface{}) error {
	log.Println("[DEBUG] addRole")

	_, err := m.(*resty.Client).R().
		SetPathParam("projectKey", projectKey).
		SetBody(role).
		Post(projectRolesUrl)

	return err
}

var updateRole = func(projectKey string, role Role, m interface{}) error {
	log.Println("[DEBUG] updateRole")

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"roleName":   role.Name,
		}).
		SetBody(role).
		Put(projectRoleUrl)

	return err
}

var deleteRoles = func(projectKey string, roles []Role, m interface{}, g *errgroup.Group) {
	log.Println("[DEBUG] deleteRoles")

	for _, role := range roles {
		projectKey, role, m := projectKey, role, m

		g.Go(func() error {
			return deleteRole(projectKey, role, m)
		})
	}
}

var deleteRole = func(projectKey string, role Role, m interface{}) error {
	log.Println("[DEBUG] deleteRole")
	log.Printf("[TRACE] %+v\n", role)

	_, err := m.(*resty.Client).R().
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
