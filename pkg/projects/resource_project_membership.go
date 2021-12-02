package projects

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Equatable interface {
	Equals(other Equatable) bool
}

// Use by both project user and project group, as they shared identical data structure
type Member struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// Use by both project user and project group, as they shared identical data structure
type Membership struct {
	Members []Member
}

func (a Member) Equals(b Member) bool {
	return a.Name == b.Name
}

func contains(as []Member, b Member) bool {
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

var membershipApply = func(predicate func(bs []Member, a Member) bool) func(as []Member, bs []Member) []Member {
	return func(as []Member, bs []Member) []Member {
		var results []Member

		// Not the most efficient way to determine the slices intersection but this suffices for the small-ish number of items
		for _, a := range as {
			if predicate(bs, a) {
				results = append(results, a)
			}
		}

		return results
	}
}

var membershipIntersection = membershipApply(func(bs []Member, a Member) bool {
	return contains(bs, a)
})

var membershipDifference = membershipApply(func(bs []Member, a Member) bool {
	return !contains(bs, a)
})

func getMembers(d *ResourceData, membershipKey string) []Member {
	var members []Member

	if v, ok := d.GetOkExists(membershipKey); ok {
		projectMemberships := v.(*schema.Set).List()
		if len(projectMemberships) == 0 {
			return members
		}

		for _, projectMembership := range projectMemberships {
			id := projectMembership.(map[string]interface{})

			member := Member{
				Name:  id["name"].(string),
				Roles: castToStringArr(id["roles"].(*schema.Set).List()),
			}
			members = append(members, member)
		}
	}

	return members
}

var unpackMembers = func(data *schema.ResourceData, membershipKey string) (string, Membership, error) {
	d := &ResourceData{data}
	projectKey := d.getString("key", false)

	membership := Membership{
		Members: getMembers(d, membershipKey),
	}

	return projectKey, membership, nil
}

var packMembers = func(d *schema.ResourceData, membershipKey string, members []Member) []error {
	log.Printf("[DEBUG] packMembership")

	setValue := mkLens(d)

	var projectMembers []interface{}

	for _, member := range members {
		log.Printf("[TRACE] %+v\n", member)
		projectMember := map[string]interface{}{
			"name":  member.Name,
			"roles": member.Roles,
		}

		projectMembers = append(projectMembers, projectMember)
	}

	log.Printf("[TRACE] %s\n", membershipKey)
	log.Printf("[TRACE] %+v\n", projectMembers)

	errors := setValue(membershipKey, projectMembers)

	return errors
}

var readMembers = func(membershipUrl string, m interface{}) ([]Member, error) {
	log.Println("[DEBUG] readMembers")

	membership := Membership{}

	_, err := m.(*resty.Client).R().SetResult(&membership).Get(membershipUrl)
	if err != nil {
		return nil, err
	}

	log.Printf("[TRACE] %+v\n", membership)

	return membership.Members, nil
}

var updateMembers = func(membershipUrl string, terraformMembership Membership, m interface{}) ([]Member, error) {
	log.Println("[DEBUG] updateMembers")
	log.Printf("[TRACE] terraformMembership: %+v\n", terraformMembership)

	projectMembers, err := readMembers(membershipUrl, m)
	log.Printf("[TRACE] projectMembers: %+v\n", projectMembers)

	membersToBeAdded := membershipDifference(terraformMembership.Members, projectMembers)
	log.Printf("[TRACE] membersToBeAdded: %+v\n", membersToBeAdded)
	membersToBeUpdated := membershipIntersection(terraformMembership.Members, projectMembers)
	log.Printf("[TRACE] membersToBeUpdated: %+v\n", membersToBeUpdated)
	membersToBeDeleted := membershipDifference(projectMembers, terraformMembership.Members)
	log.Printf("[TRACE] membersToBeDeleted: %+v\n", membersToBeDeleted)

	var errs []error

	for _, member := range append(membersToBeAdded, membersToBeUpdated...) {
		log.Printf("[TRACE] %+v\n", member)
		err := updateMember(membershipUrl, member, m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err = deleteMembers(membershipUrl, membersToBeDeleted, m)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to update members for project: %s", errs)
	}

	return readMembers(membershipUrl, m)
}

var updateMember = func(membershipUrl string, member Member, m interface{}) error {
	log.Println("[DEBUG] updateMember")

	_, err := m.(*resty.Client).R().SetBody(member).Put(membershipUrl + member.Name)

	return err
}

var deleteMembers = func(membershipUrl string, members []Member, m interface{}) error {
	log.Println("[DEBUG] deleteMembers")

	var errs []error
	for _, member := range members {
		err := deleteMember(membershipUrl, member, m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete members from project: %v", errs)
	}

	return nil
}

var deleteMember = func(membershipUrl string, member Member, m interface{}) error {
	log.Println("[DEBUG] deleteMember")
	log.Printf("[TRACE] %+v\n", member)

	_, err := m.(*resty.Client).R().Delete(membershipUrl + member.Name)
	if err != nil {
		return err
	}

	return nil
}
