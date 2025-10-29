resource "dependencytrack_managed_user" "example" {
  username = "john.doe"
  fullname = "John Doe"
  email    = "john.doe@example.com"
  password = "SecurePassword123!"
}

resource "dependencytrack_managed_user_permissions" "example" {
  user = dependencytrack_managed_user.example.id
  permissions = [
    "BOM_UPLOAD",
    "VIEW_PORTFOLIO",
    "VIEW_VULNERABILITY",
    "VULNERABILITY_ANALYSIS"
  ]
}
