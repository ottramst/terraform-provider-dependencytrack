resource "dependencytrack_tag" "production" {
  name = "production"
}

resource "dependencytrack_notification_publisher" "slack" {
  name               = "Slack Webhook"
  publisher_class    = "org.dependencytrack.notification.publisher.WebhookPublisher"
  template_mime_type = "application/json"
}

resource "dependencytrack_notification_rule" "security_alerts" {
  name      = "Security Alerts"
  scope     = "PORTFOLIO"
  publisher = dependencytrack_notification_publisher.slack.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]
}

# Limit the notification rule to projects tagged "production"
resource "dependencytrack_notification_rule_tag" "security_alerts_production" {
  tag               = dependencytrack_tag.production.name
  notification_rule = dependencytrack_notification_rule.security_alerts.uuid
}
