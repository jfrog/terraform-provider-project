resource "project_user" "myuser" {
  project_key = "myproj"
  name        = "myuser"
  roles       = ["Viewer"]
}