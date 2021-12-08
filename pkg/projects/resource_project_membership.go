package projects

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const projectMembershipsUrl = projectUrl + "/{membershipType}"
const projectMembershipUrl = projectMembershipsUrl + "/{memberName}"

const usersMembershipType = "users"
const groupssMembershipType = "groups"

// Use by both project user and project group, as they shared identical data structure
type Member struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

func (m Member) Id() string {
	return m.Name
}

func (a Member) Equals(b Identifiable) bool {
	return a.Id() == b.Id()
}

func membersToEquatables(members []Member) []Equatable {
	var equatables []Equatable

	for _, member := range members {
		equatables = append(equatables, member)
	}

	return equatables
}

func equatablesToMembers(equatables []Equatable) []Member {
	var members []Member

	for _, equatable := range equatables {
		members = append(members, equatable.(Member))
	}

	return members
}

// Use by both project user and project group, as they shared identical data structure
type Membership struct {
	Members []Member
}

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

var unpackMembers = func(data *schema.ResourceData, membershipKey string) Membership {
	d := &ResourceData{data}
	membership := Membership{
		Members: getMembers(d, membershipKey),
	}

	return membership
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

var readMembers = func(projectKey string, membershipType string, m interface{}) ([]Member, error) {
	log.Println("[DEBUG] readMembers")

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return nil, fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	membership := Membership{}

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
		   "projectKey": projectKey,
		   "membershipType": membershipType,
		}).
		SetResult(&membership).
		Get(projectMembershipsUrl)
	if err != nil {
		return nil, err
	}

	log.Printf("[TRACE] %+v\n", membership)

	return membership.Members, nil
}

var updateMembers = func(projectKey string, membershipType string, terraformMembership Membership, m interface{}) ([]Member, error) {
	log.Println("[DEBUG] updateMembers")
	log.Printf("[TRACE] terraformMembership: %+v\n", terraformMembership)

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return nil, fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	projectMembers, err := readMembers(projectKey, membershipType, m)
	log.Printf("[TRACE] projectMembers: %+v\n", projectMembers)

	membersToBeAdded := difference(membersToEquatables(terraformMembership.Members), membersToEquatables(projectMembers))
	log.Printf("[TRACE] membersToBeAdded: %+v\n", membersToBeAdded)

	membersToBeUpdated := intersection(membersToEquatables(terraformMembership.Members), membersToEquatables(projectMembers))
	log.Printf("[TRACE] membersToBeUpdated: %+v\n", membersToBeUpdated)

	membersToBeDeleted := difference(membersToEquatables(projectMembers), membersToEquatables(terraformMembership.Members))
	log.Printf("[TRACE] membersToBeDeleted: %+v\n", membersToBeDeleted)

	var errs []error

	for _, member := range append(membersToBeAdded, membersToBeUpdated...) {
		log.Printf("[TRACE] %+v\n", member)
		err := updateMember(projectKey, membershipType, member.(Member), m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err = deleteMembers(projectKey, membershipType, equatablesToMembers(membersToBeDeleted), m)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to update members for project: %s", errs)
	}

	return readMembers(projectKey, membershipType, m)
}

var updateMember = func(projectKey string, membershipType string, member Member, m interface{}) error {
	log.Println("[DEBUG] updateMember")

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
		   "projectKey": projectKey,
		   "membershipType": membershipType,
		   "memberName": member.Name,
		}).
		SetBody(member).
		Put(projectMembershipUrl)

	return err
}

var deleteMembers = func(projectKey string, membershipType string, members []Member, m interface{}) error {
	log.Println("[DEBUG] deleteMembers")

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	var errs []error
	for _, member := range members {
		err := deleteMember(projectKey, membershipType, member, m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete members from project: %v", errs)
	}

	return nil
}

var deleteMember = func(projectKey string, membershipType string, member Member, m interface{}) error {
	log.Println("[DEBUG] deleteMember")
	log.Printf("[TRACE] %+v\n", member)

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
		   "projectKey": projectKey,
		   "membershipType": membershipType,
		   "memberName": member.Name,
		}).
		Delete(projectMembershipUrl)
	if err != nil {
		return err
	}

	return nil
}
