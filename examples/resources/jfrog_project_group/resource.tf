resource "jfrog_project_group" "mygroup" {
  project_key = "myproj"
  name        = "mygroup"
  roles       = ["Viewer"]
}
