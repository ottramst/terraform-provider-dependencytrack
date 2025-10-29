# Look up a team by UUID
data "dependencytrack_team" "by_uuid" {
  uuid = "00000000-0000-0000-0000-000000000000"
}

# Look up a team by name
data "dependencytrack_team" "by_name" {
  name = "Security Team"
}
