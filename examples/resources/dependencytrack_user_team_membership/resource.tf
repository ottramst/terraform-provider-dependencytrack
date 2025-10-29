resource "dependencytrack_team" "developers" {
  name = "Developers"
}

resource "dependencytrack_managed_user" "john_doe" {
  username = "john.doe"
  fullname = "John Doe"
  email    = "john.doe@example.com"
  password = "SecurePassword123!"
}

resource "dependencytrack_user_team_membership" "john_developers" {
  username = dependencytrack_managed_user.john_doe.username
  team     = dependencytrack_team.developers.id
}