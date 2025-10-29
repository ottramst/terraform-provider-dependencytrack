resource "dependencytrack_team" "example" {
  name = "Security Team"
}

resource "dependencytrack_team_permissions" "example" {
  team = dependencytrack_team.example.id
  permissions = [
    "BOM_UPLOAD",
    "VIEW_PORTFOLIO",
    "VIEW_VULNERABILITY",
    "VULNERABILITY_ANALYSIS"
  ]
}