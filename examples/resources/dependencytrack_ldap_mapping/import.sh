# LDAP mappings can be imported using the format: team_uuid/mapping_uuid
# The distinguished name is not part of the import ID because it commonly contains '/'.
terraform import dependencytrack_ldap_mapping.example 00000000-0000-0000-0000-000000000001/00000000-0000-0000-0000-000000000002
