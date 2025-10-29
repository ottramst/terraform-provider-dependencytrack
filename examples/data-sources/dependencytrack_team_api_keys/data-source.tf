data "dependencytrack_team" "automation" {
  name = "Automation Team"
}

data "dependencytrack_team_api_keys" "automation_keys" {
  team = data.dependencytrack_team.automation.id
}

# Output the masked keys
output "api_keys" {
  value = [
    for key in data.dependencytrack_team_api_keys.automation_keys.api_keys : {
      public_id  = key.public_id
      comment    = key.comment
      masked_key = key.masked_key
      legacy     = key.legacy
    }
  ]
}