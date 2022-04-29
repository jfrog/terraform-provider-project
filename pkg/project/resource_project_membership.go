package project

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jfrog/terraform-provider-shared/util"
	"golang.org/x/sync/errgroup"
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

func (a Member) Equals(b Equatable) bool {
	return a.Id() == b.Id()
}

// Use by both project user and project group, as they shared identical data structure
type Membership struct {
	Members []Member
}

func getMembers(d *util.ResourceData, membershipKey string) []Member {
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
				Roles: util.CastToStringArr(id["roles"].(*schema.Set).List()),
			}
			members = append(members, member)
		}
	}

	return members
}

var unpackMembers = func(data *schema.ResourceData, membershipKey string) Membership {
	d := &util.ResourceData{data}
	membership := Membership{
		Members: getMembers(d, membershipKey),
	}

	return membership
}

var packMembers = func(ctx context.Context, d *schema.ResourceData, membershipKey string, members []Member) []error {
	tflog.Debug(ctx, "packMembership")

	setValue := util.MkLens(d)

	var projectMembers []interface{}

	for _, member := range members {
		tflog.Trace(ctx, fmt.Sprintf("%+v\n", member))
		projectMember := map[string]interface{}{
			"name":  member.Name,
			"roles": member.Roles,
		}

		projectMembers = append(projectMembers, projectMember)
	}

	tflog.Trace(ctx, fmt.Sprintf("%s\n", membershipKey))
	tflog.Trace(ctx, fmt.Sprintf("%+v\n", projectMembers))

	errors := setValue(membershipKey, projectMembers)

	return errors
}

var readMembers = func(ctx context.Context, projectKey string, membershipType string, m interface{}) ([]Member, error) {
	tflog.Debug(ctx, "readMembers")

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return nil, fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	membership := Membership{}

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
			"projectKey":     projectKey,
			"membershipType": membershipType,
		}).
		SetResult(&membership).
		Get(projectMembershipsUrl)
	if err != nil {
		return nil, err
	}

	tflog.Trace(ctx, fmt.Sprintf("readMembers: %+v\n", membership))

	return membership.Members, nil
}

var updateMembers = func(ctx context.Context, projectKey string, membershipType string, terraformMembership Membership, m interface{}) ([]Member, error) {
	tflog.Debug(ctx, "updateMembers")
	tflog.Trace(ctx, fmt.Sprintf("terraformMembership.Members: %+v\n", terraformMembership.Members))

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return nil, fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	projectMembers, err := readMembers(ctx, projectKey, membershipType, m)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch memberships for project: %s", err)
	}
	tflog.Trace(ctx, fmt.Sprintf("projectMembers: %+v\n", projectMembers))

	terraformMembersSet := SetFromSlice(terraformMembership.Members)
	projectMembersSet := SetFromSlice(projectMembers)
	membersToBeAdded := terraformMembersSet.Difference(projectMembersSet)
	tflog.Trace(ctx, fmt.Sprintf("membersToBeAdded: %+v\n", membersToBeAdded))
	membersToBeUpdated := terraformMembersSet.Intersection(projectMembersSet)
	tflog.Trace(ctx, fmt.Sprintf("membersToBeUpdated: %+v\n", membersToBeUpdated))
	membersToBeDeleted := projectMembersSet.Difference(terraformMembersSet)
	tflog.Trace(ctx, fmt.Sprintf("membersToBeDeleted: %+v\n", membersToBeDeleted))

	g := new(errgroup.Group)

	for _, member := range append(membersToBeAdded, membersToBeUpdated...) {
		projectKey, membershipType, member, m := projectKey, membershipType, member, m

		g.Go(func() error {
			return updateMember(ctx, projectKey, membershipType, member, m)
		})
	}

	deleteMembers(ctx, projectKey, membershipType, membersToBeDeleted, m, g)

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to update memberships for project: %v", err)
	}

	return readMembers(ctx, projectKey, membershipType, m)
}

var updateMember = func(ctx context.Context, projectKey string, membershipType string, member Member, m interface{}) error {
	tflog.Debug(ctx, "updateMember")
	tflog.Trace(ctx, fmt.Sprintf("member: %v", member))

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
			"projectKey":     projectKey,
			"membershipType": membershipType,
			"memberName":     member.Name,
		}).
		SetBody(member).
		Put(projectMembershipUrl)

	return err
}

var deleteMembers = func(ctx context.Context, projectKey string, membershipType string, members []Member, m interface{}, g *errgroup.Group) {
	tflog.Debug(ctx, "deleteMembers")

	for _, member := range members {
		projectKey, membershipType, member, m := projectKey, membershipType, member, m

		g.Go(func() error {
			return deleteMember(ctx, projectKey, membershipType, member, m)
		})
	}
}

var deleteMember = func(ctx context.Context, projectKey string, membershipType string, member Member, m interface{}) error {
	tflog.Debug(ctx, "deleteMember")
	tflog.Trace(ctx, fmt.Sprintf("%+v\n", member))

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return fmt.Errorf("Invalid membershipType: %s", membershipType)
	}

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
			"projectKey":     projectKey,
			"membershipType": membershipType,
			"memberName":     member.Name,
		}).
		Delete(projectMembershipUrl)
	if err != nil {
		return err
	}

	return nil
}
