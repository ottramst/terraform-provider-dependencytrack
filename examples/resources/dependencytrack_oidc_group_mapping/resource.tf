resource "dependencytrack_team" "platform" {
  name = "Platform Engineering"
}

resource "dependencytrack_oidc_group" "platform" {
  name = "platform-engineers"
}

resource "dependencytrack_oidc_group_mapping" "platform" {
  group = dependencytrack_oidc_group.platform.id
  team  = dependencytrack_team.platform.id
}
