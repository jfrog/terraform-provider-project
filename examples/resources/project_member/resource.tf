resource "project_member" "myuser" {
  project_key = "myproj"
  name        = "myuser"
  roles       = ["Viewer"]
}