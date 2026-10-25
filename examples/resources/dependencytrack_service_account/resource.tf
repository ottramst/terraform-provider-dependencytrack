# A service account for a CI pipeline (Dependency-Track 5.2+)
resource "dependencytrack_service_account" "ci" {
  name  = "ci-pipeline"
  email = "platform-team@example.com"
}

# Grant permissions directly, keyed by the account's svc: username
resource "dependencytrack_managed_user_permissions" "ci" {
  user        = dependencytrack_service_account.ci.username
  permissions = ["BOM_UPLOAD", "VIEW_PORTFOLIO"]
}

# Or through a team
resource "dependencytrack_team" "automation" {
  name = "CI Automation"
}

resource "dependencytrack_user_team_membership" "ci" {
  username = dependencytrack_service_account.ci.username
  team     = dependencytrack_team.automation.id
}
