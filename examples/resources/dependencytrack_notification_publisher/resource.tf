# Webhook notification publisher
resource "dependencytrack_notification_publisher" "webhook" {
  name               = "Slack Webhook"
  description        = "Sends notifications to Slack"
  publisher_class    = "org.dependencytrack.notification.publisher.WebhookPublisher"
  template_mime_type = "application/json"
  template = jsonencode({
    text = "New vulnerability detected in {{project.name}}"
  })
}

# Console notification publisher (minimal example)
resource "dependencytrack_notification_publisher" "console" {
  name               = "Console Logger"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

# Email notification publisher
resource "dependencytrack_notification_publisher" "email" {
  name               = "Email Notifications"
  description        = "Sends email notifications for critical vulnerabilities"
  publisher_class    = "org.dependencytrack.notification.publisher.SendMailPublisher"
  template_mime_type = "text/plain"
  template           = "Project: {{project.name}}\nVulnerability: {{vulnerability.vulnId}}"
}
