# Share repository with a project
resource "jfrog_project_share_repository" "myprojectsharerepo" {
  repo_key           = "myrepo-generic-local"
  target_project_key = "myproj"
}

# Share repository with multiple projects
resource "jfrog_project_share_repository" "share_repo" {
  count = 3

  repo_key = artifactory_local_generic_repository.repo.key
  target_project_key = element(
    [
      jfrog_project.project_name_1.key,
      jfrog_project.project_name_2.key,
      jfrog_project.project_name_3.key
    ],
    count.index
  )
  read_only = true
}
