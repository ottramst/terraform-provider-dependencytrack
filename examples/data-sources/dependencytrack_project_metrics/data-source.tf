data "dependencytrack_project" "web_app" {
  name    = "Web Application"
  version = "1.0.0"
}

# Fetch the latest metrics snapshot of the project
data "dependencytrack_project_metrics" "web_app" {
  project = data.dependencytrack_project.web_app.id
}

output "web_app_critical_vulnerabilities" {
  value = data.dependencytrack_project_metrics.web_app.critical
}
