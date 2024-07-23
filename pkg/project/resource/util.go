package project

import (
	"fmt"
	"regexp"

	"github.com/go-resty/resty/v2"
	"github.com/jfrog/terraform-provider-shared/util"
	"github.com/samber/lo"
)

type Equatable interface {
	util.Identifiable
	Equals(other Equatable) bool
}

func RetryOnSpecificMsgBody(matchString string) func(response *resty.Response, err error) bool {
	return func(response *resty.Response, err error) bool {
		return regexp.MustCompile(matchString).MatchString(string(response.Body()[:]))
	}
}

type ProjectError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ProjectError) String() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Message)
}

type ProjectErrorsResponse struct {
	Errors []ProjectError `json:"errors"`
}

func (r ProjectErrorsResponse) String() string {
	errs := lo.Reduce(r.Errors, func(err string, item ProjectError, _ int) string {
		if err == "" {
			return item.String()
		} else {
			return fmt.Sprintf("%s, %s", err, item.String())
		}
	}, "")
	return errs
}

const ProjectRepositoryStatusEndpoint = "access/api/v1/projects/_/repositories/{repo_key}"

type ProjectRepositoryStatusAPIModel struct {
	ResourceName          string   `json:"resource_name"`
	Environments          []string `json:"environments"`
	SharedWithProjects    []string `json:"shared_with_projects"`
	SharedWithAllProjects bool     `json:"shared_with_all_projects"`
	SharedReadOnly        bool     `json:"shared_read_only"`
	AssignedTo            string   `json:"assigned_to"`
}
