resource "project_role" "myrole" {
    name = "myrole"
    type = "CUSTOM"
    project_key = project.myproject.key
    
    environments = ["DEV"]
    actions = ["READ_REPOSITORY", "ANNOTATE_REPOSITORY"]
}
