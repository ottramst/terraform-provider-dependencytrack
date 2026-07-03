resource "dependencytrack_license" "example" {
  license_id   = "Acme-1.0"
  name         = "Acme Corporation License 1.0"
  text         = "Permission is hereby granted..."
  comment      = "Internal license used by Acme projects"
  osi_approved = false
  fsf_libre    = false
  see_also = [
    "https://acme.example.com/license",
  ]
}
