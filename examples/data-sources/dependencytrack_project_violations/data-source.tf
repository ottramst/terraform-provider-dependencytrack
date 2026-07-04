data "dependencytrack_project" "web_app" {
  name    = "Web Application"
  version = "1.0.0"
}

# Fetch the unsuppressed policy violations of the project
data "dependencytrack_project_violations" "web_app" {
  project = data.dependencytrack_project.web_app.id
}

# Include suppressed violations as well
data "dependencytrack_project_violations" "web_app_all" {
  project    = data.dependencytrack_project.web_app.id
  suppressed = true
}

output "web_app_violated_policies" {
  value = toset(data.dependencytrack_project_violations.web_app.violations[*].policy_name)
}
