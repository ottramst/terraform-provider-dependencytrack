resource "dependencytrack_service_account" "ci" {
  name = "ci-pipeline"
}

resource "dependencytrack_workload_identity_provider" "github_actions" {
  name     = "github-actions"
  type     = "OIDC"
  issuer   = "https://token.actions.githubusercontent.com"
  audience = "https://dtrack.example.com"
}

# Let workflows of acme/app on main act as the ci-pipeline service account.
resource "dependencytrack_workload_identity_binding" "app_main" {
  service_account = dependencytrack_service_account.ci.name
  provider_name   = dependencytrack_workload_identity_provider.github_actions.name
  subject         = "repo:acme/app:*"
  condition       = "claims.ref == \"refs/heads/main\""
}
