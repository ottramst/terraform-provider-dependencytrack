resource "dependencytrack_project" "example" {
  name        = "My Application"
  version     = "1.0.0"
  description = "My application project"
  group       = "com.example"
  classifier  = "APPLICATION"
  active      = true
}