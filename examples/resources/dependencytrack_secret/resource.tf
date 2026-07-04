# A secret referenced by name from extension configs or repository credentials
resource "dependencytrack_secret" "github_token" {
  name        = "github-advisories-token"
  value       = var.github_token
  description = "GitHub access token for the GitHub Advisories vulnerability source"
}

# Reference the secret by name wherever a secret reference is expected
resource "dependencytrack_extension_config" "github_advisories" {
  extension_point = "vuln-data-source"
  extension       = "github"

  config = jsonencode({
    enabled          = true
    aliasSyncEnabled = true
    apiUrl           = "https://api.github.com/graphql"
    apiToken         = dependencytrack_secret.github_token.name
  })
}
