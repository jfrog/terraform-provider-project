# Share repository with a project
resource "project_share_repository" "myprojectsharerepo" {
  repo_key = "myrepo-generic-local"
  target_project_key = "myproj"
}

# Share repository with multiple projects
resource "project_share_repository" "share_repo" {
  count = 3

  repo_key = artifactory_local_generic_repository.repo.key
  target_project_key = element(
    [
      project.project_name_1.key,
      project.project_name_2.key,
      project.project_name_3.key
    ],
    count.index
  )
  read_only  = true
}
