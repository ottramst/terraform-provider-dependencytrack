resource "dependencytrack_project" "web_app" {
  name    = "Web Application"
  version = "1.0.0"
}

resource "dependencytrack_policy" "license_policy" {
  name            = "License Compliance Policy"
  operator        = "ANY"
  violation_state = "FAIL"
}

resource "dependencytrack_project_policy" "web_app_license" {
  policy  = dependencytrack_policy.license_policy.id
  project = dependencytrack_project.web_app.id
}
