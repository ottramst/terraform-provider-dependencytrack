# Enable the Trivy vulnerability analyzer against a self-hosted Trivy server
resource "dependencytrack_secret" "trivy_token" {
  name  = "trivy-api-token"
  value = var.trivy_api_token
}

resource "dependencytrack_extension_config" "trivy" {
  extension_point = "vuln-analyzer"
  extension       = "trivy"

  config = jsonencode({
    enabled       = true
    apiUrl        = "http://trivy-server.example.svc.cluster.local:8080"
    apiToken      = dependencytrack_secret.trivy_token.name
    scanLibrary   = true
    scanOs        = true
    ignoreUnfixed = false
  })
}

# Enable the OSV vulnerability data source for selected ecosystems
resource "dependencytrack_extension_config" "osv" {
  extension_point = "vuln-data-source"
  extension       = "osv"

  config = jsonencode({
    enabled                     = true
    aliasSyncEnabled            = false
    incrementalMirroringEnabled = true
    dataUrl                     = "https://storage.googleapis.com/osv-vulnerabilities"
    ecosystems                  = ["Maven", "npm"]
  })
}

# Configure SMTP via the email notification publisher
resource "dependencytrack_secret" "smtp_password" {
  name  = "smtp-password"
  value = var.smtp_password
}

resource "dependencytrack_extension_config" "email" {
  extension_point = "notification-publisher"
  extension       = "email"

  config = jsonencode({
    enabled         = true
    host            = "smtp.example.com"
    port            = 587
    username        = "no-reply@example.com"
    password        = dependencytrack_secret.smtp_password.name
    senderAddress   = "no-reply@example.com"
    sslEnabled      = false
    startTlsEnabled = true
  })
}
