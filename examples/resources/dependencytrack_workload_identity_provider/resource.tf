# Trust GitHub Actions OIDC tokens. The signing keys are resolved from the
# issuer's discovery document.
resource "dependencytrack_workload_identity_provider" "github_actions" {
  name     = "github-actions"
  type     = "OIDC"
  issuer   = "https://token.actions.githubusercontent.com"
  audience = "https://dtrack.example.com"
}

# An issuer whose public keys are provided inline
resource "dependencytrack_workload_identity_provider" "internal" {
  name                     = "internal-ci"
  type                     = "OIDC"
  issuer                   = "https://ci.internal.example.com"
  audience                 = "dependency-track"
  jwks                     = file("${path.module}/internal-ci-jwks.json")
  session_lifetime_seconds = 900
}
