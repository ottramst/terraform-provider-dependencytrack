# Fetch all repositories
data "dependencytrack_repositories" "all" {}

# Fetch only Maven repositories
data "dependencytrack_repositories" "maven" {
  type = "MAVEN"
}
