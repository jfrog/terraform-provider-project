package project

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
	d := &ResourceData{data}

	var repoKeys []RepoKey

	if v, ok := d.GetOkExists("repos"); ok {
		for _, key := range castToStringArr(v.(*schema.Set).List()) {
			repoKeys = append(repoKeys, RepoKey(key))
		}
	}

	return repoKeys
}

var packRepos = func(d *schema.ResourceData, repoKeys []RepoKey) []error {
	log.Printf("[DEBUG] packRepos")
	log.Printf("[TRACE] repos: %+v\n", repoKeys)

	setValue := mkLens(d)

	errors := setValue("repos", repoKeys)

	return errors
}

var readRepos = func(projectKey string, m interface{}) ([]RepoKey, error) {
	log.Println("[DEBUG] readRepos")

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

	log.Printf("[TRACE] artifactoryRepos: %+v\n", artifactoryRepos)

	var repoKeys []RepoKey

	for _, artifactoryRepo := range artifactoryRepos {
		repoKeys = append(repoKeys, RepoKey(artifactoryRepo.Key))
	}

	return repoKeys, nil
}

var updateRepos = func(projectKey string, terraformRepoKeys []RepoKey, m interface{}) ([]RepoKey, error) {
	log.Println("[DEBUG] updateRepos")
	log.Printf("[TRACE] terraformRepoKeys: %+v\n", terraformRepoKeys)

	projectRepoKeys, err := readRepos(projectKey, m)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos for project: %s", err)
	}
	log.Printf("[TRACE] projectRepoKeys: %+v\n", projectRepoKeys)

	terraformRepoKeysSet := SetFromSlice(terraformRepoKeys)
	projectRepoKeysSet := SetFromSlice(projectRepoKeys)

	repoKeysToBeAdded := terraformRepoKeysSet.Difference(projectRepoKeysSet)
	log.Printf("[TRACE] repoKeysToBeAdded: %+v\n", repoKeysToBeAdded)

	repoKeysToBeDeleted := projectRepoKeysSet.Difference(terraformRepoKeysSet)
	log.Printf("[TRACE] repoKeysToBeDeleted: %+v\n", repoKeysToBeDeleted)

	g := new(errgroup.Group)

	for _, repoKey := range repoKeysToBeAdded {
		projectKey, repoKey, m := projectKey, repoKey, m

		g.Go(func() error {
			return addRepo(projectKey, repoKey, m)
		})
	}

	deleteRepos(projectKey, repoKeysToBeDeleted, m, g)
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to update repos for project: %s", err)
	}

	return readRepos(projectKey, m)
}

var addRepo = func(projectKey string, repoKey RepoKey, m interface{}) error {
	log.Println("[DEBUG] addRepo")

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

var deleteRepos = func(projectKey string, repoKeys []RepoKey, m interface{}, g *errgroup.Group) {
	log.Println("[DEBUG] deleteRepos")

	for _, repoKey := range repoKeys {
		projectKey, repoKey, m := projectKey, repoKey, m

		g.Go(func() error {
			return deleteRepo(projectKey, repoKey, m)
		})
	}
}

var deleteRepo = func(projectKey string, repoKey RepoKey, m interface{}) error {
	log.Println("[DEBUG] deleteRepo")

	_, err := m.(*resty.Client).
		R().
		AddRetryCondition(retryOnSpecificMsgBody("A timeout occurred")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is down")).
		AddRetryCondition(retryOnSpecificMsgBody("Web server is returning an unknown error")).
		SetPathParam("repoKey", string(repoKey)).
		Delete(projectsUrl + "/_/attach/repositories/{repoKey}")

	return err
}
