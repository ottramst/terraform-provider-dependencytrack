# Look up a notification publisher by UUID
data "dependencytrack_notification_publisher" "by_uuid" {
  uuid = "00000000-0000-0000-0000-000000000000"
}

# Look up a notification publisher by name
data "dependencytrack_notification_publisher" "by_name" {
  name = "Slack Webhook"
}