resource "dependencytrack_notification_rule" "example" {
  name      = "Example Rule"
  scope     = "PORTFOLIO"
  notify_on = ["NEW_VULNERABILITY"]
  publisher = dependencytrack_notification_publisher.slack.id
}

resource "dependencytrack_project" "example" {
  name = "Example Project"
}

resource "dependencytrack_notification_rule_project" "example" {
  rule    = dependencytrack_notification_rule.example.id
  project = dependencytrack_project.example.id
}
