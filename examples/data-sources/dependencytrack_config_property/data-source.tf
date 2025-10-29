# Look up a config property by group name and property name
data "dependencytrack_config_property" "example" {
  group_name = "general"
  name       = "base.url"
}