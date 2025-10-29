resource "dependencytrack_policy" "example" {
  name             = "Critical Vulnerabilities"
  operator         = "ALL"
  violation_state  = "FAIL"
  include_children = true
  global           = true

  conditions = [
    {
      subject  = "SEVERITY"
      operator = "IS"
      value    = "CRITICAL"
    },
    {
      subject  = "LICENSE"
      operator = "IS"
      value    = "GPL-3.0"
    }
  ]
}