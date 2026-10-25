# API Key Authentication
provider "dependencytrack" {
  endpoint = "https://dtrack.example.com"
  api_key  = "your-api-key-here"
}

# Username/Password Authentication
provider "dependencytrack" {
  endpoint = "https://dtrack.example.com"
  username = "admin"
  password = "admin123"
}

# Workload identity federation (Dependency-Track 5.2+): exchange a workload's
# identity token, here a GitHub Actions OIDC token, for a session of a
# service account. See dependencytrack_workload_identity_binding.
provider "dependencytrack" {
  endpoint = "https://dtrack.example.com"

  workload_identity = {
    provider_name   = "github-actions"
    service_account = "ci-pipeline"
    token           = var.github_oidc_token
  }
}
