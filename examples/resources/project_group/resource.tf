resource "project_group" "mygroup" {
  project_key = "myproj"
  name        = "mygroup"
  roles       = ["Viewer"]
}