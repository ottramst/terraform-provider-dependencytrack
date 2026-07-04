# Look up a license by its SPDX-style license ID
data "dependencytrack_license" "apache" {
  license_id = "Apache-2.0"
}
