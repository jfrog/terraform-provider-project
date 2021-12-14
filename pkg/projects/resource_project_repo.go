package projects

import (
	"fmt"
	"log"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Repo struct {
	Name string
}

func (this Repo) Id() string {
	return this.Name
}

func (this Repo) Equals(other Identifiable) bool {
	return this.Id() == other.Id()
}

func reposToEquatables(repos []Repo) []Equatable {
	var equatables []Equatable

	for _, repo := range repos {
		equatables = append(equatables, repo)
	}

	return equatables
}

func equatablesToRepos(equatables []Equatable) []Repo {
	var repos []Repo

	for _, equatable := range equatables {
		repos = append(repos, equatable.(Repo))
	}

	return repos
}

var unpackRepos = func(data *schema.ResourceData) []Repo {
	d := &ResourceData{data}

	var repos []Repo

	if v, ok := d.GetOkExists("repo"); ok {
		projectRepos := v.(*schema.Set).List()
		if len(projectRepos) == 0 {
			return repos
		}

		for _, projectRepo := range projectRepos {
			id := projectRepo.(map[string]interface{})

			repo := Repo{
				Name:      id["name"].(string),
			}
			repos = append(repos, repo)
		}
	}

	return repos
}

var packRepos = func(d *schema.ResourceData, repos []Repo) []error {
	log.Printf("[DEBUG] packRepos")

	setValue := mkLens(d)

	var projectRepos []interface{}

	for _, repo := range repos {
		log.Printf("[TRACE] %+v\n", repo)
		projectRepo := map[string]interface{}{
			"name":  repo.Name,
		}

		projectRepos = append(projectRepos, projectRepo)
	}

	log.Printf("[TRACE] %+v\n", projectRepos)

	errors := setValue("repo", projectRepos)

	return errors
}

var readRepos = func(projectKey string, m interface{}) ([]Repo, error) {
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

	var repos []Repo
	for _, artifactoryRepo := range artifactoryRepos {
		repo := Repo{
			Name: artifactoryRepo.Key,
		}
		repos = append(repos, repo)
	}
	log.Printf("[TRACE] repos: %+v\n", repos)

	return repos, nil
}

var updateRepos = func(projectKey string, terraformRepos []Repo, m interface{}) ([]Repo, error) {
	log.Println("[DEBUG] updateRepos")
	log.Printf("[TRACE] terraformRepos: %+v\n", terraformRepos)

	projectRepos, err := readRepos(projectKey, m)
	log.Printf("[TRACE] projectRepos: %+v\n", projectRepos)

	reposToBeAdded := difference(reposToEquatables(terraformRepos), reposToEquatables(projectRepos))
	log.Printf("[TRACE] reposToBeAdded: %+v\n", reposToBeAdded)

	reposToBeDeleted := difference(reposToEquatables(projectRepos), reposToEquatables(terraformRepos))
	log.Printf("[TRACE] reposToBeDeleted: %+v\n", reposToBeDeleted)

	var errs []error

	for _, repo := range reposToBeAdded {
		log.Printf("[TRACE] %+v\n", repo)
		err := addRepo(projectKey, repo.(Repo), m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err = deleteRepos(projectKey, equatablesToRepos(reposToBeDeleted), m)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to update repos for project: %s", errs)
	}

	return readRepos(projectKey, m)
}

var addRepo = func(projectKey string, repo Repo, m interface{}) error {
	log.Println("[DEBUG] addRepo")

	_, err := m.(*resty.Client).R().
		SetPathParams(map[string]string{
			"projectKey": projectKey,
			"repoName":   repo.Name,
		}).
		SetQueryParam("force", "true").
		Put(projectsUrl + "/_/attach/repositories/{repoName}/{projectKey}")

	return err
}

var deleteRepos = func(projectKey string, repos []Repo, m interface{}) error {
	log.Println("[DEBUG] deleteRepos")

	var errs []error
	for _, repo := range repos {
		err := deleteRepo(projectKey, repo, m)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete repos from project: %v", errs)
	}

	return nil
}

var deleteRepo = func(projectKey string, repo Repo, m interface{}) error {
	log.Println("[DEBUG] deleteRepo")
	log.Printf("[TRACE] %+v\n", repo)

	_, err := m.(*resty.Client).R().
		SetPathParam("repoName", repo.Name).
		Delete(projectsUrl + "/_/attach/repositories/{repoName}")

	if err != nil {
		return err
	}

	return nil
}
