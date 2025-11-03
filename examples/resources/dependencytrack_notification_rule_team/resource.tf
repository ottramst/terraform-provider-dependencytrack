resource "dependencytrack_notification_rule" "example" {
  name      = "Example Rule"
  scope     = "PORTFOLIO"
  notify_on = ["NEW_VULNERABILITY"]
  publisher = dependencytrack_notification_publisher.email.uuid
}

resource "dependencytrack_team" "example" {
  name = "Example Team"
}

resource "dependencytrack_notification_rule_team" "example" {
  rule_uuid = dependencytrack_notification_rule.example.uuid
  team_uuid = dependencytrack_team.example.uuid
}
