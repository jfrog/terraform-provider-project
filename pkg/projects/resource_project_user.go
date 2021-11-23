package projects

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type User struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

type Comparable interface {
	Hash() string
	Equals(other Comparable) bool
}

func (user User) Hash() string {
	return user.Name
}

func (a User) Equals(b User) bool {
	return a.Hash() == b.Hash()
}

func contains(as []User, b User) bool {
	log.Printf("[DEBUG] contains")
	log.Printf("[TRACE] as: %+v\n", as)
	log.Printf("[TRACE] b: %+v\n", b)

	for _, a := range as {
		log.Printf("[TRACE] a: %+v\n", a)
		log.Printf("[TRACE] a.Equals(b): %+v\n", a.Equals(b))
		if a.Equals(b) {
			return true
		}
	}
	return false
}

// Not the most efficient way to determine the slices intersection but this suffices for the small-ish number of items
var usersIntersection = func(as []User, bs []User) []User {
	log.Printf("[DEBUG] usersIntersection")
	log.Printf("[TRACE] as: %+v\n", as)
	log.Printf("[TRACE] bs: %+v\n", bs)

	intersection := make([]User, 0)

	for _, a := range as {
		if contains(bs, a) {
			intersection = append(intersection, a)
		}
	}

	return intersection
}

// Not the most efficient way to determine the slices differences but this suffices for the small-ish number of items
var usersDifference = func(as []User, bs []User) []User {
	difference := make([]User, 0)

	for _, a := range as {
		if !contains(bs, a) {
			difference = append(difference, a)
		}
	}

	return difference
}

type Users struct {
	Members []User
}

const projectUsersUrl = "/access/api/v1/projects/%s/users/"

var unpackUsers = func(data *schema.ResourceData) (string, Users, error) {
	d := &ResourceData{data}
	projectKey := d.getString("key", false)

	users := Users{
		Members: make([]User, 0),
	}

	if v, ok := d.GetOkExists("user"); ok {
		projectUsers := v.(*schema.Set).List()
		if len(projectUsers) == 0 {
			return projectKey, users, nil
		}

		for _, projectUser := range projectUsers {
			id := projectUser.(map[string]interface{})

			user := User{
				Name:  id["name"].(string),
				Roles: castToStringArr(id["roles"].(*schema.Set).List()),
			}
			users.Members = append(users.Members, user)
		}
	}

	return projectKey, users, nil
}

var packUsers = func(d *schema.ResourceData, key string, users *[]User) []error {
	log.Printf("[DEBUG] packUsers")

	setValue := mkLens(d)

	var projectUsers []interface{}

	for _, user := range *users {
		log.Printf("[TRACE] %+v\n", user)
		projectUser := map[string]interface{}{
			"name":  user.Name,
			"roles": user.Roles,
		}

		projectUsers = append(projectUsers, projectUser)
	}

	log.Printf("[TRACE] %s\n", key)
	log.Printf("[TRACE] %+v\n", projectUsers)

	errors := setValue(key, projectUsers)

	return errors
}

var getProjectsUsersUrl = func(projectKey string, id string) string {
	return fmt.Sprintf(projectUsersUrl, projectKey) + id
}

var readUsers = func(projectKey string, m interface{}) ([]User, error) {
	log.Println("[DEBUG] readUsers")

	users := Users{}

	_, err := m.(*resty.Client).R().SetResult(&users).Get(fmt.Sprintf(projectUsersUrl, projectKey))
	if err != nil {
		return nil, err
	}

	log.Printf("[TRACE] %+v\n", users)

	return users.Members, nil
}

var updateUsers = func(projectKey string, terraformUsers Users, m interface{}) ([]User, error) {
	log.Println("[DEBUG] updateUsers")
	log.Printf("[TRACE] terraformUsers: %+v\n", terraformUsers)

	projectUsers, err := readUsers(projectKey, m)
	log.Printf("[TRACE] projectUsers: %+v\n", projectUsers)

	usersToBeAdded := usersDifference(terraformUsers.Members, projectUsers)
	log.Printf("[TRACE] usersToBeAdded: %+v\n", usersToBeAdded)
	usersToBeUpdated := usersIntersection(terraformUsers.Members, projectUsers)
	log.Printf("[TRACE] usersToBeUpdated: %+v\n", usersToBeUpdated)
	usersToBeDeleted := usersDifference(projectUsers, terraformUsers.Members)
	log.Printf("[TRACE] usersToBeDeleted: %+v\n", usersToBeDeleted)

	var errs []error

	for _, user := range append(usersToBeAdded, usersToBeUpdated...) {
		log.Printf("[TRACE] %+v\n", user)
		err := updateUser(projectKey, &user, m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err = deleteUsers(projectKey, usersToBeDeleted, m)
	if err != nil {
		errs = append(errs, err)
	}

	if errs != nil && len(errs) > 0 {
		return nil, fmt.Errorf("failed to update users for project: %s", fmt.Sprintf("%s", errs))
	}

	return readUsers(projectKey, m)
}

var updateUser = func(projectKey string, user *User, m interface{}) error {
	log.Println("[DEBUG] updateUser")

	_, err := m.(*resty.Client).R().SetBody(user).Put(getProjectsUsersUrl(projectKey, user.Name))

	return err
}

var deleteUsers = func(projectKey string, users []User, m interface{}) error {
	log.Println("[DEBUG] deleteUsers")

	var errs []error
	for _, user := range users {
		fmt.Printf("[TRACE] %+v\n", user)

		_, err := m.(*resty.Client).R().Delete(getProjectsUsersUrl(projectKey, user.Name))
		if err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil && len(errs) > 0 {
		return fmt.Errorf("failed to delete users from project: %s", fmt.Sprintf("%s", errs))
	}

	return nil
}
