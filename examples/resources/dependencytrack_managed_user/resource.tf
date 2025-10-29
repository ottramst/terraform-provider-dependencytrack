resource "dependencytrack_managed_user" "example" {
  username = "johndoe"
  fullname = "John Doe"
  email    = "john.doe@example.com"
  password = "SecureP@ssw0rd123"

  suspended             = false
  force_password_change = false
  non_expiry_password   = false
}