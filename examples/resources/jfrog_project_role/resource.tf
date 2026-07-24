resource "jfrog_project_role" "myrole" {
  name        = "myrole"
  type        = "CUSTOM"
  project_key = jfrog_project.myproject.key

  environments = ["DEV"]
  actions      = ["READ_REPOSITORY", "ANNOTATE_REPOSITORY"]
}
