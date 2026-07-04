resource "dependencytrack_tag" "production" {
  name = "production"
}

resource "dependencytrack_policy" "license_policy" {
  name            = "License Compliance Policy"
  operator        = "ANY"
  violation_state = "FAIL"
}

# Limit the policy to projects tagged "production"
resource "dependencytrack_policy_tag" "license_policy_production" {
  tag    = dependencytrack_tag.production.name
  policy = dependencytrack_policy.license_policy.id
}
