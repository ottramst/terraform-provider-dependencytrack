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

# Notification rule for specific projects
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

  projects = [
    dependencytrack_project.web_app.id
  ]

  message = "Alert for project: {{project.name}}"
}

# Notification rule for specific teams
resource "dependencytrack_team" "security_team" {
  name = "Security Team"
}

resource "dependencytrack_notification_rule" "team_notifications" {
  name      = "Security Team Alerts"
  scope     = "SYSTEM"
  publisher = dependencytrack_notification_publisher.slack.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]

  teams = [
    dependencytrack_team.security_team.id
  ]
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
