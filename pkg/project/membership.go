package project

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/jfrog/terraform-provider-shared/util/sdk"
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

func getMembers(d *sdk.ResourceData, membershipKey string) []Member {
	var members []Member

	if v, ok := d.GetOk(membershipKey); ok {
		projectMemberships := v.(*schema.Set).List()
		if len(projectMemberships) == 0 {
			return members
		}

		for _, projectMembership := range projectMemberships {
			id := projectMembership.(map[string]interface{})

			member := Member{
				Name:  id["name"].(string),
				Roles: sdk.CastToStringArr(id["roles"].(*schema.Set).List()),
			}
			members = append(members, member)
		}
	}

	return members
}

var unpackMembers = func(data *schema.ResourceData, membershipKey string) Membership {
	d := &sdk.ResourceData{ResourceData: data}
	membership := Membership{
		Members: getMembers(d, membershipKey),
	}

	return membership
}

var packMembers = func(ctx context.Context, d *schema.ResourceData, membershipKey string, members []Member) []error {
	tflog.Debug(ctx, "packMembership")

	setValue := sdk.MkLens(d)

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
		return nil, fmt.Errorf("invalid membershipType: %s", membershipType)
	}

	membership := Membership{}

	var projectError ProjectErrorsResponse
	resp, err := m.(util.ProviderMetadata).Client.R().
		SetPathParams(map[string]string{
			"projectKey":     projectKey,
			"membershipType": membershipType,
		}).
		SetResult(&membership).
		SetError(&projectError).
		Get(projectMembershipsUrl)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("%s", projectError.String())
	}

	tflog.Trace(ctx, fmt.Sprintf("readMembers: %+v\n", membership))

	return membership.Members, nil
}

var updateMembers = func(ctx context.Context, projectKey string, membershipType string, terraformMembership Membership, m interface{}) ([]Member, error) {
	tflog.Debug(ctx, "updateMembers")
	tflog.Trace(ctx, fmt.Sprintf("terraformMembership.Members: %+v\n", terraformMembership.Members))

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return nil, fmt.Errorf("invalid membershipType: %s", membershipType)
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

	for _, member := range append(membersToBeAdded, membersToBeUpdated...) {
		err := updateMember(ctx, projectKey, membershipType, member, m)
		if err != nil {
			return nil, fmt.Errorf("failed to update members %s: %s", member, err)
		}
	}

	deleteErr := deleteMembers(ctx, projectKey, membershipType, membersToBeDeleted, m)
	if deleteErr != nil {
		return nil, fmt.Errorf("failed to delete members for project: %s", deleteErr)
	}

	return readMembers(ctx, projectKey, membershipType, m)
}

var updateMember = func(ctx context.Context, projectKey string, membershipType string, member Member, m interface{}) error {
	tflog.Debug(ctx, "updateMember")
	tflog.Trace(ctx, fmt.Sprintf("member: %v", member))

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return fmt.Errorf("invalid membershipType: %s", membershipType)
	}

	var projectError ProjectErrorsResponse
	resp, err := m.(util.ProviderMetadata).Client.R().
		SetPathParams(map[string]string{
			"projectKey":     projectKey,
			"membershipType": membershipType,
			"memberName":     member.Name,
		}).
		SetBody(member).
		SetError(&projectError).
		Put(projectMembershipUrl)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", projectError.String())
	}

	return err
}

var deleteMembers = func(ctx context.Context, projectKey string, membershipType string, members []Member, m interface{}) error {
	tflog.Debug(ctx, "deleteMembers")

	for _, member := range members {
		err := deleteMember(ctx, projectKey, membershipType, member, m)
		if err != nil {
			return fmt.Errorf("failed to delete %s %s: %s", membershipType, member, err)
		}
	}

	return nil
}

var deleteMember = func(ctx context.Context, projectKey string, membershipType string, member Member, m interface{}) error {
	tflog.Debug(ctx, "deleteMember")
	tflog.Trace(ctx, fmt.Sprintf("%+v\n", member))

	if membershipType != usersMembershipType && membershipType != groupssMembershipType {
		return fmt.Errorf("invalid membershipType: %s", membershipType)
	}

	var projectError ProjectErrorsResponse
	resp, err := m.(util.ProviderMetadata).Client.R().
		SetPathParams(map[string]string{
			"projectKey":     projectKey,
			"membershipType": membershipType,
			"memberName":     member.Name,
		}).
		SetError(&projectError).
		Delete(projectMembershipUrl)
	if err != nil {
		return err
	}
	if resp.IsError() && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("%s", projectError.String())
	}

	return nil
}
