# Basic notification rule for vulnerabilities
resource "dependencytrack_notification_publisher" "slack" {
  name               = "Slack Webhook"
  publisher_class    = "org.dependencytrack.notification.publisher.WebhookPublisher"
  template_mime_type = "application/json"
}

resource "dependencytrack_notification_rule" "vulnerability_alerts" {
  name               = "Critical Vulnerability Alerts"
  scope              = "PORTFOLIO"
  notification_level = "ERROR"
  publisher          = dependencytrack_notification_publisher.slack.id

  notify_on = [
    "NEW_VULNERABILITY",
    "NEW_VULNERABLE_DEPENDENCY"
  ]

  enabled                = true
  notify_children        = true
  log_successful_publish = false
}

# Notification rule with project association (use separate resource)
resource "dependencytrack_project" "web_app" {
  name    = "Web Application"
  version = "1.0.0"
}

resource "dependencytrack_notification_rule" "project_specific" {
  name      = "Web App Notifications"
  scope     = "PORTFOLIO"
  publisher = dependencytrack_notification_publisher.slack.id

  notify_on = [
    "NEW_VULNERABILITY",
    "POLICY_VIOLATION"
  ]
}

resource "dependencytrack_notification_rule_project" "web_app" {
  rule    = dependencytrack_notification_rule.project_specific.id
  project = dependencytrack_project.web_app.id
}

# System-level notification rule
resource "dependencytrack_notification_rule" "system_alerts" {
  name               = "System Configuration Alerts"
  scope              = "SYSTEM"
  notification_level = "WARNING"
  publisher          = dependencytrack_notification_publisher.slack.id

  notify_on = [
    "CONFIGURATION",
    "DATASOURCE_MIRRORING"
  ]
}

# Dependency-Track 5.2+: signed webhook. The receiver can verify each payload's
# HMAC-SHA256 signature with the shared secret.
resource "dependencytrack_secret" "webhook_signing" {
  name  = "webhook-signing-secret"
  value = var.webhook_signing_secret
}

resource "dependencytrack_notification_rule" "signed_webhook" {
  name      = "Signed Webhook"
  scope     = "PORTFOLIO"
  publisher = data.dependencytrack_notification_publisher.webhook.id

  notify_on = ["NEW_VULNERABILITY"]

  publisher_config = jsonencode({
    destinationUrl = "https://hooks.example.com/dependency-track"
    signingSecret  = dependencytrack_secret.webhook_signing.name
  })
}

data "dependencytrack_notification_publisher" "webhook" {
  name = "Webhook"
}

variable "webhook_signing_secret" {
  type      = string
  sensitive = true
}
