resource "dependencytrack_notification_rule" "example" {
  name      = "Example Rule"
  scope     = "PORTFOLIO"
  notify_on = ["NEW_VULNERABILITY"]
  publisher = dependencytrack_notification_publisher.email.id
}

resource "dependencytrack_team" "example" {
  name = "Example Team"
}

resource "dependencytrack_notification_rule_team" "example" {
  rule = dependencytrack_notification_rule.example.id
  team = dependencytrack_team.example.id
}
