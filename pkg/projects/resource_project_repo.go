package projects

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type RepoKey string

func (this RepoKey) Id() string {
	return string(this)
}

func (this RepoKey) Equals(other Identifiable) bool {
	return this == other
}

func repoKeysToEquatables(repoKeys []RepoKey) []Equatable {
	var equatables []Equatable

	for _, repoKey := range repoKeys {
		equatables = append(equatables, repoKey)
	}

	return equatables
}

func equatablesToRepoKeys(equatables []Equatable) []RepoKey {
	var repoKeys []RepoKey

	for _, equatable := range equatables {
		repoKeys = append(repoKeys, equatable.(RepoKey))
	}

	return repoKeys
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
	log.Printf("[TRACE] projectRepoKeys: %+v\n", projectRepoKeys)

	repoKeysToBeAdded := difference(repoKeysToEquatables(terraformRepoKeys), repoKeysToEquatables(projectRepoKeys))
	log.Printf("[TRACE] repoKeysToBeAdded: %+v\n", repoKeysToBeAdded)

	repoKeysToBeDeleted := difference(repoKeysToEquatables(projectRepoKeys), repoKeysToEquatables(terraformRepoKeys))
	log.Printf("[TRACE] repoKeysToBeDeleted: %+v\n", repoKeysToBeDeleted)

	var errs []error

	for _, repoKey := range repoKeysToBeAdded {
		err := addRepo(projectKey, repoKey.(RepoKey), m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err = deleteRepos(projectKey, equatablesToRepoKeys(repoKeysToBeDeleted), m)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to update repos for project: %s", errs)
	}

	return readRepos(projectKey, m)
}

var addRepo = func(projectKey string, repoKey RepoKey, m interface{}) error {
	log.Println("[DEBUG] addRepo")

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"repoKey":    string(repoKey),
		}).
		SetQueryParam("force", "true").
		Put(projectsUrl + "/_/attach/repositories/{repoKey}/{projectKey}")

	return err
}

var deleteRepos = func(projectKey string, repoKeys []RepoKey, m interface{}) error {
	log.Println("[DEBUG] deleteRepos")

	var errs []error
	for _, repoKey := range repoKeys {
		err := deleteRepo(projectKey, repoKey, m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete repos from project: %v", errs)
	}

	return nil
}

var deleteRepo = func(projectKey string, repoKey RepoKey, m interface{}) error {
	log.Println("[DEBUG] deleteRepo")
	log.Printf("[TRACE] %+v\n", repoKey)

	_, err := m.(*resty.Client).R().
		SetPathParam("repoKey", string(repoKey)).
		Delete(projectsUrl + "/_/attach/repositories/{repoKey}")

	if err != nil {
		return err
	}

	return nil
}
