resource "dependencytrack_license_group" "copyleft" {
  name = "Copyleft"
}

resource "dependencytrack_license" "custom" {
  license_id = "my-custom-license"
  name       = "My Custom License"
}

resource "dependencytrack_license_group_license" "copyleft_custom" {
  license_group = dependencytrack_license_group.copyleft.id
  license       = dependencytrack_license.custom.uuid
}
