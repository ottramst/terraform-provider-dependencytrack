resource "dependencytrack_notification_rule" "example" {
  name      = "Example Rule"
  scope     = "PORTFOLIO"
  notify_on = ["NEW_VULNERABILITY"]
  publisher = dependencytrack_notification_publisher.slack.uuid
}

resource "dependencytrack_project" "example" {
  name = "Example Project"
}

resource "dependencytrack_notification_rule_project" "example" {
  rule_uuid    = dependencytrack_notification_rule.example.uuid
  project_uuid = dependencytrack_project.example.uuid
}
