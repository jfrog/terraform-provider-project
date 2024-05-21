package project

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const projectMembershipsUrl = ProjectUrl + "/{membershipType}"
const projectMembershipUrl = projectMembershipsUrl + "/{memberName}"

const usersMembershipType = "users"
const groupsMembershipType = "groups"

// Use by both project user and project group, as they shared identical data structure
type MemberAPIModel struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

func (m MemberAPIModel) Id() string {
	return m.Name
}

func (a MemberAPIModel) Equals(b Equatable) bool {
	return a.Id() == b.Id()
}

// Use by both project user and project group, as they shared identical data structure
type MembershipAPIModel struct {
	Members []MemberAPIModel
}

var readMembers = func(ctx context.Context, projectKey, membershipType string, client *resty.Client) ([]MemberAPIModel, error) {
	tflog.Debug(ctx, "readMembers")

	if membershipType != usersMembershipType && membershipType != groupsMembershipType {
		return nil, fmt.Errorf("invalid membershipType: %s", membershipType)
	}

	var membership MembershipAPIModel
	var projectError ProjectErrorsResponse
	resp, err := client.R().
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

var updateMembers = func(ctx context.Context, projectKey, membershipType string, members []MemberAPIModel, client *resty.Client) ([]MemberAPIModel, error) {
	tflog.Debug(ctx, "updateMembers")
	tflog.Trace(ctx, fmt.Sprintf("terraformMembership.Members: %+v\n", members))

	if membershipType != usersMembershipType && membershipType != groupsMembershipType {
		return nil, fmt.Errorf("invalid membershipType: %s", membershipType)
	}

	projectMembers, err := readMembers(ctx, projectKey, membershipType, client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch memberships for project: %s", err)
	}
	tflog.Trace(ctx, fmt.Sprintf("projectMembers: %+v\n", projectMembers))

	terraformMembersSet := SetFromSlice(members)
	projectMembersSet := SetFromSlice(projectMembers)

	membersToBeAdded := terraformMembersSet.Difference(projectMembersSet)
	tflog.Trace(ctx, fmt.Sprintf("membersToBeAdded: %+v\n", membersToBeAdded))
	membersToBeUpdated := terraformMembersSet.Intersection(projectMembersSet)
	tflog.Trace(ctx, fmt.Sprintf("membersToBeUpdated: %+v\n", membersToBeUpdated))
	membersToBeDeleted := projectMembersSet.Difference(terraformMembersSet)
	tflog.Trace(ctx, fmt.Sprintf("membersToBeDeleted: %+v\n", membersToBeDeleted))

	for _, member := range append(membersToBeAdded, membersToBeUpdated...) {
		err := updateMember(ctx, projectKey, membershipType, member, client)
		if err != nil {
			return nil, fmt.Errorf("failed to update members %s: %s", member, err)
		}
	}

	deleteErr := deleteMembers(ctx, projectKey, membershipType, membersToBeDeleted, client)
	if deleteErr != nil {
		return nil, fmt.Errorf("failed to delete members for project: %s", deleteErr)
	}

	return readMembers(ctx, projectKey, membershipType, client)
}

var updateMember = func(ctx context.Context, projectKey, membershipType string, member MemberAPIModel, client *resty.Client) error {
	tflog.Debug(ctx, "updateMember")
	tflog.Trace(ctx, fmt.Sprintf("member: %v", member))

	if membershipType != usersMembershipType && membershipType != groupsMembershipType {
		return fmt.Errorf("invalid membershipType: %s", membershipType)
	}

	var projectError ProjectErrorsResponse
	resp, err := client.R().
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

var deleteMembers = func(ctx context.Context, projectKey, membershipType string, members []MemberAPIModel, client *resty.Client) error {
	tflog.Debug(ctx, "deleteMembers")

	for _, member := range members {
		err := deleteMember(ctx, projectKey, membershipType, member, client)
		if err != nil {
			return fmt.Errorf("failed to delete %s %s: %s", membershipType, member, err)
		}
	}

	return nil
}

var deleteMember = func(ctx context.Context, projectKey, membershipType string, member MemberAPIModel, client *resty.Client) error {
	tflog.Debug(ctx, "deleteMember")
	tflog.Trace(ctx, fmt.Sprintf("%+v\n", member))

	if membershipType != usersMembershipType && membershipType != groupsMembershipType {
		return fmt.Errorf("invalid membershipType: %s", membershipType)
	}

	var projectError ProjectErrorsResponse
	resp, err := client.R().
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
