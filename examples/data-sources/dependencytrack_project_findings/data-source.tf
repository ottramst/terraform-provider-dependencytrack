data "dependencytrack_project" "web_app" {
  name    = "Web Application"
  version = "1.0.0"
}

# Fetch the unsuppressed vulnerability findings of the project
data "dependencytrack_project_findings" "web_app" {
  project = data.dependencytrack_project.web_app.id
}

# Include suppressed findings as well
data "dependencytrack_project_findings" "web_app_all" {
  project    = data.dependencytrack_project.web_app.id
  suppressed = true
}

output "web_app_vulnerability_ids" {
  value = toset(data.dependencytrack_project_findings.web_app.findings[*].vuln_id)
}
