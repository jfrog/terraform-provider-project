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

	_, err := m.(*resty.Client).R().
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

	g := new(errgroup.Group)

	for _, repoKey := range repoKeysToBeAdded {
		projectKey, repoKey, m := projectKey, repoKey, m

		g.Go(func() error {
			return addRepo(ctx, projectKey, repoKey, m)
		})
	}

	deleteRepos(ctx, projectKey, repoKeysToBeDeleted, m, g)
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to update repos for project: %s", err)
	}

	return readRepos(ctx, projectKey, m)
}

var addRepo = func(ctx context.Context, projectKey string, repoKey RepoKey, m interface{}) error {
	tflog.Debug(ctx, "addRepo")

	_, err := m.(*resty.Client).
		R().
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

var deleteRepos = func(ctx context.Context, projectKey string, repoKeys []RepoKey, m interface{}, g *errgroup.Group) {
	tflog.Debug(ctx, "deleteRepos")

	for _, repoKey := range repoKeys {
		projectKey, repoKey, m := projectKey, repoKey, m

		g.Go(func() error {
			return deleteRepo(ctx, projectKey, repoKey, m)
		})
	}
}

var deleteRepo = func(ctx context.Context, projectKey string, repoKey RepoKey, m interface{}) error {
	tflog.Debug(ctx, "deleteRepo")

	_, err := m.(*resty.Client).
		R().
		AddRetryCondition(retryOnSpecificMsgBody("A timeout occurred")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is down")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is returning an unknown error")).
		SetPathParam("repoKey", string(repoKey)).
		Delete(projectsUrl + "/_/attach/repositories/{repoKey}")

	return err
}
