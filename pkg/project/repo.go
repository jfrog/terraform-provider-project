package project

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/jfrog/terraform-provider-shared/util"
)

type RepoKey string

func (this RepoKey) Id() string {
	return string(this)
}

func (this RepoKey) Equals(other Equatable) bool {
	return this == other
}

var unpackRepos = func(data *schema.ResourceData) []RepoKey {
	d := &util.ResourceData{data}

	var repoKeys []RepoKey

	if v, ok := d.GetOkExists("repos"); ok {
		for _, key := range util.CastToStringArr(v.(*schema.Set).List()) {
			repoKeys = append(repoKeys, RepoKey(key))
		}
	}

	return repoKeys
}

var packRepos = func(ctx context.Context, d *schema.ResourceData, repoKeys []RepoKey) []error {
	tflog.Debug(ctx, "packRepos")
	tflog.Trace(ctx, fmt.Sprintf("repos: %+v\n", repoKeys))

	setValue := util.MkLens(d)

	errors := setValue("repos", repoKeys)

	return errors
}

var readRepos = func(ctx context.Context, projectKey string, m interface{}) ([]RepoKey, error) {
	tflog.Debug(ctx, "readRepos")

	type ArtifactoryRepo struct {
		Key string
	}

	artifactoryRepos := []ArtifactoryRepo{}

	_, err := m.(util.ProvderMetadata).Client.R().
		SetPathParam("projectKey", projectKey).
		SetResult(&artifactoryRepos).
		Get("/artifactory/api/repositories?project={projectKey}")

	if err != nil {
		return nil, err
	}

	tflog.Trace(ctx, fmt.Sprintf("artifactoryRepos: %+v\n", artifactoryRepos))

	var repoKeys []RepoKey

	for _, artifactoryRepo := range artifactoryRepos {
		repoKeys = append(repoKeys, RepoKey(artifactoryRepo.Key))
	}

	return repoKeys, nil
}

var updateRepos = func(ctx context.Context, projectKey string, terraformRepoKeys []RepoKey, m interface{}) ([]RepoKey, error) {
	tflog.Debug(ctx, "updateRepos")
	tflog.Trace(ctx, fmt.Sprintf("terraformRepoKeys: %+v\n", terraformRepoKeys))

	projectRepoKeys, err := readRepos(ctx, projectKey, m)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos for project: %s", err)
	}
	tflog.Trace(ctx, fmt.Sprintf("projectRepoKeys: %+v\n", projectRepoKeys))

	terraformRepoKeysSet := SetFromSlice(terraformRepoKeys)
	projectRepoKeysSet := SetFromSlice(projectRepoKeys)

	repoKeysToBeAdded := terraformRepoKeysSet.Difference(projectRepoKeysSet)
	tflog.Trace(ctx, fmt.Sprintf("repoKeysToBeAdded: %+v\n", repoKeysToBeAdded))

	repoKeysToBeDeleted := projectRepoKeysSet.Difference(terraformRepoKeysSet)
	tflog.Trace(ctx, fmt.Sprintf("repoKeysToBeDeleted: %+v\n", repoKeysToBeDeleted))

	addErr := addRepos(ctx, projectKey, repoKeysToBeAdded, m)
	if addErr != nil {
		return nil, fmt.Errorf("failed to add repos for project: %s", addErr)
	}

	deleteErr := deleteRepos(ctx, projectKey, repoKeysToBeDeleted, m)
	if deleteErr != nil {
		return nil, fmt.Errorf("failed to delete repos for project: %s", deleteErr)
	}

	return readRepos(ctx, projectKey, m)
}

var addRepos = func(ctx context.Context, projectKey string, repoKeys []RepoKey, m interface{}) error {
	tflog.Debug(ctx, fmt.Sprintf("addRepos: %s", repoKeys))

	for _, repoKey := range repoKeys {
		err := addRepo(ctx, projectKey, repoKey, m)
		if err != nil {
			return fmt.Errorf("failed to add repo %s: %s", repoKey, err)
		}
	}

	return nil
}

var addRepo = func(ctx context.Context, projectKey string, repoKey RepoKey, m interface{}) error {
	tflog.Debug(ctx, fmt.Sprintf("addRepo: %s", repoKey))

	_, err := m.(util.ProvderMetadata).Client.R().
		AddRetryCondition(retryOnSpecificMsgBody("A timeout occurred")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is down")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is returning an unknown error")).
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"repoKey":    string(repoKey),
		}).
		SetQueryParam("force", "true").
		Put(projectsUrl + "/_/attach/repositories/{repoKey}/{projectKey}")

	return err
}

var deleteRepos = func(ctx context.Context, projectKey string, repoKeys []RepoKey, m interface{}) error {
	tflog.Debug(ctx, fmt.Sprintf("deleteRepos: %s", repoKeys))

	for _, repoKey := range repoKeys {
		err := deleteRepo(ctx, projectKey, repoKey, m)
		if err != nil {
			return fmt.Errorf("failed to delete repo %s: %s", repoKey, err)
		}
	}

	return nil
}

var deleteRepo = func(ctx context.Context, projectKey string, repoKey RepoKey, m interface{}) error {
	tflog.Debug(ctx, fmt.Sprintf("deleteRepo: %s", repoKey))

	type Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	type ErrorResponse struct {
		Errors []Error `json:"errors"`
	}

	var errorResp ErrorResponse

	resp, err := m.(util.ProvderMetadata).Client.R().
		AddRetryCondition(retryOnSpecificMsgBody("A timeout occurred")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is down")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is returning an unknown error")).
		SetPathParam("repoKey", string(repoKey)).
		SetError(&errorResp).
		Delete(projectsUrl + "/_/attach/repositories/{repoKey}")

	// Ignore 404 NOT_FOUND error when unassigning repo from project
	// Possible that repo was deleted out-of-band from TF
	if resp.StatusCode() == http.StatusNotFound && len(errorResp.Errors) > 0 {
		for _, error := range errorResp.Errors {
			if error.Code == "NOT_FOUND" {
				tflog.Warn(ctx, fmt.Sprintf("failed to unassign repo: %s", error.Message))
				return nil
			}
		}
	}

	return err
}
