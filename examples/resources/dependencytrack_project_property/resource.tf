resource "dependencytrack_project" "example" {
  name    = "Web Application"
  version = "1.0.0"
}

resource "dependencytrack_project_property" "example" {
  project     = dependencytrack_project.example.id
  group       = "integrations"
  name        = "cost-center"
  value       = "CC-1234"
  type        = "STRING"
  description = "Cost center owning this project"
}
