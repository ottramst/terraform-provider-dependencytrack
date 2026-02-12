# Look up a project by ID (UUID)
data "dependencytrack_project" "by_uuid" {
  id = "00000000-0000-0000-0000-000000000000"
}

# Look up a project by name and version
data "dependencytrack_project" "by_name_version" {
  name    = "My Application"
  version = "1.0.0"
}