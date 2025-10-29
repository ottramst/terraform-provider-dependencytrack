resource "dependencytrack_team" "automation" {
  name = "Automation Team"
}

resource "dependencytrack_team_api_key" "ci_cd" {
  team    = dependencytrack_team.automation.id
  comment = "CI/CD Pipeline API Key"
}

# The API key value is only available on creation
output "api_key" {
  value     = dependencytrack_team_api_key.ci_cd.key
  sensitive = true
}