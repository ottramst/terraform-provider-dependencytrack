resource "dependencytrack_team" "developers" {
  name = "Developers"
}

resource "dependencytrack_ldap_mapping" "developers" {
  team = dependencytrack_team.developers.id
  dn   = "CN=Developers,OU=Groups,DC=example,DC=com"
}
