# Look up a license group by name
data "dependencytrack_license_group" "by_name" {
  name = "Copyleft"
}

# Look up a license group by UUID
data "dependencytrack_license_group" "by_id" {
  id = "00000000-0000-0000-0000-000000000000"
}
