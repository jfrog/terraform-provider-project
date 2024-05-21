package project

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"
)

type ArtifactoryRepo struct {
	Key string
}

var readRepos = func(ctx context.Context, projectKey string, client *resty.Client) ([]string, error) {
	tflog.Debug(ctx, "readRepos")

	var artifactoryRepos []ArtifactoryRepo
	var projectError ProjectErrorsResponse
	resp, err := client.R().
		SetPathParam("projectKey", projectKey).
		SetResult(&artifactoryRepos).
		SetError(&projectError).
		Get("/artifactory/api/repositories?project={projectKey}")

	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("%s", projectError.String())
	}

	tflog.Trace(ctx, fmt.Sprintf("artifactoryRepos: %+v\n", artifactoryRepos))

	var repoKeys []string
	for _, artifactoryRepo := range artifactoryRepos {
		repoKeys = append(repoKeys, artifactoryRepo.Key)
	}

	return repoKeys, nil
}

var updateRepos = func(ctx context.Context, projectKey string, terraformRepoKeys []string, client *resty.Client) ([]string, error) {
	tflog.Debug(ctx, "updateRepos")
	tflog.Trace(ctx, fmt.Sprintf("terraformRepoKeys: %+v\n", terraformRepoKeys))

	projectRepoKeys, err := readRepos(ctx, projectKey, client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos for project: %s", err)
	}
	tflog.Trace(ctx, fmt.Sprintf("projectRepoKeys: %+v\n", projectRepoKeys))

	repoKeysToBeAdded, repoKeysToBeDeleted := lo.Difference(terraformRepoKeys, projectRepoKeys)
	tflog.Trace(ctx, fmt.Sprintf("repoKeysToBeAdded: %+v\n", repoKeysToBeAdded))
	tflog.Trace(ctx, fmt.Sprintf("repoKeysToBeDeleted: %+v\n", repoKeysToBeDeleted))

	addErr := addRepos(ctx, projectKey, repoKeysToBeAdded, client)
	if addErr != nil {
		return nil, fmt.Errorf("failed to add repos for project: %s", addErr)
	}

	deleteErr := deleteRepos(ctx, repoKeysToBeDeleted, client)
	if deleteErr != nil {
		return nil, fmt.Errorf("failed to delete repos for project: %s", deleteErr)
	}

	return readRepos(ctx, projectKey, client)
}

var addRepos = func(ctx context.Context, projectKey string, repoKeys []string, client *resty.Client) error {
	tflog.Debug(ctx, fmt.Sprintf("addRepos: %s", repoKeys))

	req := client.R().
		AddRetryCondition(RetryOnSpecificMsgBody("A timeout occurred")).
		AddRetryCondition(RetryOnSpecificMsgBody("Web server is down")).
		AddRetryCondition(RetryOnSpecificMsgBody("Web server is returning an unknown error"))

	for _, repoKey := range repoKeys {
		err := addRepo(ctx, projectKey, repoKey, req)
		if err != nil {
			return fmt.Errorf("failed to add repo %s: %s", repoKey, err)
		}
	}

	return nil
}

var addRepo = func(ctx context.Context, projectKey, repoKey string, req *resty.Request) error {
	tflog.Debug(ctx, fmt.Sprintf("addRepo: %s", repoKey))

	var projectError ProjectErrorsResponse
	resp, err := req.
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"repoKey":    string(repoKey),
		}).
		SetQueryParam("force", "true").
		SetError(&projectError).
		Put(ProjectsUrl + "/_/attach/repositories/{repoKey}/{projectKey}")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("%s", projectError.String())
	}

	return err
}

var deleteRepos = func(ctx context.Context, repoKeys []string, client *resty.Client) error {
	tflog.Debug(ctx, fmt.Sprintf("deleteRepos: %s", repoKeys))

	req := client.R().
		AddRetryCondition(RetryOnSpecificMsgBody("A timeout occurred")).
		AddRetryCondition(RetryOnSpecificMsgBody("Web server is down")).
		AddRetryCondition(RetryOnSpecificMsgBody("Web server is returning an unknown error"))

	for _, repoKey := range repoKeys {
		err := deleteRepo(ctx, repoKey, req)
		if err != nil {
			return fmt.Errorf("failed to delete repo %s: %s", repoKey, err)
		}
	}

	return nil
}

var deleteRepo = func(ctx context.Context, repoKey string, req *resty.Request) error {
	tflog.Debug(ctx, fmt.Sprintf("deleteRepo: %s", repoKey))

	var projectError ProjectErrorsResponse
	resp, err := req.
		SetPathParam("repoKey", repoKey).
		SetError(&projectError).
		Delete(ProjectsUrl + "/_/attach/repositories/{repoKey}")

	if err != nil {
		return err
	}

	// Ignore 404 NOT_FOUND error when unassigning repo from project
	// Possible that repo was deleted out-of-band from TF
	if resp.StatusCode() == http.StatusNotFound && len(projectError.Errors) > 0 {
		for _, e := range projectError.Errors {
			if e.Code == "NOT_FOUND" {
				tflog.Warn(ctx, fmt.Sprintf("failed to unassign repo: %s", e.Message))
				return nil
			}
		}
	} else if resp.IsError() {
		return fmt.Errorf("%s", projectError.String())
	}

	return nil
}
