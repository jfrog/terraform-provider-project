package projects

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var validRoleTypes = []string{
	"PREDEFINED",
	"CUSTOM",
}

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

func (a Role) Equals(b Role) bool {
	return a.Name == b.Name
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

var readRoles = func(m interface{}) ([]Role, error) {
	log.Println("[DEBUG] readRoles")

	roles := []Role{}

	_, err := m.(*resty.Client).R().SetResult(&roles).Get(projectRolesUrl)
	if err != nil {
		return nil, err
	}

	log.Printf("[TRACE] %+v\n", roles)

	return roles, nil
}

var deleteRoles = func(roles []Role, m interface{}) error {
	log.Println("[DEBUG] deleteRoles")

	var errs []error
	for _, role := range roles {
		err := deleteRole(role, m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete roles from project: %v", errs)
	}

	return nil
}

var deleteRole = func(role Role, m interface{}) error {
	log.Println("[DEBUG] deleteRole")
	log.Printf("[TRACE] %+v\n", role)

	_, err := m.(*resty.Client).R().Delete(projectRolesUrl + role.Name)
	if err != nil {
		return err
	}

	return nil
}
