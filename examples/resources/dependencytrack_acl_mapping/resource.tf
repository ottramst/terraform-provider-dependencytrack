resource "dependencytrack_team" "security_team" {
  name = "Security Team"
}

resource "dependencytrack_project" "web_app" {
  name    = "Web Application"
  version = "1.0.0"
}

resource "dependencytrack_acl_mapping" "security_to_web_app" {
  team    = dependencytrack_team.security_team.id
  project = dependencytrack_project.web_app.id
}