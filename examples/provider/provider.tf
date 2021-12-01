provider "project" {
  url = "${var.artifactory_url}/projects"
  access_token = "${var.artifactory_access_token}"
}
