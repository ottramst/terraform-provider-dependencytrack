resource "dependencytrack_repository" "example" {
  type       = "MAVEN"
  identifier = "internal-maven"
  url        = "https://repo.example.com/maven"
  enabled    = true
  internal   = true
}

# A repository that requires authentication
resource "dependencytrack_repository" "private_npm" {
  type                    = "NPM"
  identifier              = "private-npm"
  url                     = "https://npm.example.com"
  authentication_required = true
  username                = "npm-user"
  password                = var.npm_password
}
