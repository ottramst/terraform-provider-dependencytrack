resource "dependencytrack_service_account" "ci" {
  name = "ci-pipeline"
}

# Rotate the key every 30 days, ahead of its 60-day expiry.
resource "time_rotating" "ci_key" {
  rotation_days = 30
}

resource "dependencytrack_service_account_api_key" "ci" {
  service_account = dependencytrack_service_account.ci.name
  comment         = "GitHub Actions"
  expires_in_days = 60

  lifecycle {
    replace_triggered_by = [time_rotating.ci_key]
  }
}

output "ci_api_key" {
  value     = dependencytrack_service_account_api_key.ci.key
  sensitive = true
}
