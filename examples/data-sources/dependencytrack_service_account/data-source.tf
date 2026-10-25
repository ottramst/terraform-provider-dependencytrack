data "dependencytrack_service_account" "ci" {
  name = "ci-pipeline"
}

output "ci_permissions" {
  value = data.dependencytrack_service_account.ci.permissions
}
