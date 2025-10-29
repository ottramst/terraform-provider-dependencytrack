## v0.2.1

BREAKING CHANGES:

* resource/user_team_membership: The `team_uuid` attribute has been renamed to `team` for consistency with other resources

ENHANCEMENTS:

* resource/user_team_membership: Added example files for resource usage and import
* docs: Updated documentation to reflect the `team` attribute naming

## v0.2.0

FEATURES:

* **New Resource:** `dependencytrack_user_team_membership` - Manage user memberships in teams in Dependency-Track

BREAKING CHANGES:

* resource/managed_user: The `fullname` field is now required as mandated by the Dependency-Track API

ENHANCEMENTS:

* resource/user_team_membership: Supports full CRUD operations for user team memberships
* resource/user_team_membership: Supports import using the format `username/team`
* resource/user_team_membership: Works with managed, LDAP, and OIDC users
* tests: Added acceptance tests for `dependencytrack_user_team_membership` resource using API key authentication

BUG FIXES:

* build: Fixed artifact path in GitHub workflow to use correct binary name `terraform-provider-dependencytrack`
* build: Updated .gitignore to reflect correct binary name
* module: Renamed Go module from `terraform-provider-dependency-track` to `terraform-provider-dependencytrack` for consistency
* docs: Fixed GitHub repository URLs in CHANGELOG.md to use correct repository name

## v0.1.0

FEATURES:

* **New Data Source:** `dependencytrack_config_property` - Retrieve configuration property information
* **New Data Source:** `dependencytrack_managed_user` - Retrieve managed user information
* **New Data Source:** `dependencytrack_policy` - Retrieve policy information by UUID
* **New Data Source:** `dependencytrack_project` - Retrieve project information by UUID or by name and version
* **New Data Source:** `dependencytrack_team_api_keys` - Retrieve all API keys for a team
* **New Data Source:** `dependencytrack_team` - Retrieve team information by name or UUID
* **New Resource:** `dependencytrack_acl_mapping` - Manage ACL mappings between teams and projects in Dependency-Track
* **New Resource:** `dependencytrack_config_property` - Manage Dependency-Track configuration properties
* **New Resource:** `dependencytrack_managed_user_permissions` - Manage permissions for managed users in Dependency-Track
* **New Resource:** `dependencytrack_managed_user` - Manage managed users in Dependency-Track
* **New Resource:** `dependencytrack_policy` - Manage Dependency-Track policies with support for policy conditions
* **New Resource:** `dependencytrack_project` - Manage Dependency-Track projects
* **New Resource:** `dependencytrack_team_api_key` - Manage API keys for Dependency-Track teams
* **New Resource:** `dependencytrack_team_permissions` - Manage team permissions in Dependency-Track
* **New Resource:** `dependencytrack_team` - Manage teams in Dependency-Track

ENHANCEMENTS:

* ci: Added CI/CD workflows for build, test, lint, and documentation generation
* data source/policy: Retrieves complete policy information including all conditions
* data source/project: Supports lookup by UUID or by name and version combination
* data source/team_api_keys: Retrieves all API keys for a team including metadata (public_id, comment, masked_key, legacy flag)
* resource/acl_mapping: Supports full CRUD operations for ACL mappings between teams and projects
* resource/acl_mapping: Supports import using the format `team_uuid/project_uuid`
* resource/config_property: Configuration properties are adopted into Terraform state and can be managed without creating or deleting them in Dependency-Track
* resource/config_property: Supports import using the format `group_name/property_name`
* resource/managed_user_permissions: Automatically reconciles permission additions and removals during updates
* resource/managed_user_permissions: Manages multiple permissions for a managed user in a single resource
* resource/managed_user_permissions: Supports import using username
* resource/policy: Supports full CRUD operations for policies including all core attributes
* resource/policy: Supports global policies and hierarchical policies with an include_children flag
* resource/policy: Supports import using UUID
* resource/policy: Supports policy conditions for defining policy criteria (subject, operator, value)
* resource/project: Supports full CRUD operations for projects including all core attributes
* resource/project: Supports import using UUID
* resource/team_api_key: API key value is only available on creation and marked as sensitive
* resource/team_api_key: Supports generating new API keys for teams with optional comments
* resource/team_api_key: Supports updating the comment field for existing API keys
* resource/team_permissions: Automatically reconciles permission additions and removals during updates
* resource/team_permissions: Manages multiple permissions for a team in a single resource
* resource/team_permissions: Supports import using team UUID
* tests: Added acceptance tests for API key authentication on `dependencytrack_config_property` resource and data source
* tests: Added acceptance tests for API key authentication on `dependencytrack_managed_user` resource and data source
* tests: Added acceptance tests for `dependencytrack_acl_mapping` resource using API key authentication
* tests: Added acceptance tests for `dependencytrack_managed_user_permissions` resource using API key authentication
* tests: Added acceptance tests for `dependencytrack_policy` resource and data source using API key authentication
* tests: Added acceptance tests for `dependencytrack_project` resource and data source using API key authentication
* tests: Added acceptance tests for `dependencytrack_team_api_key` resource using API key authentication
* tests: Added acceptance tests for `dependencytrack_team_api_keys` data source using API key authentication
* tests: Added acceptance tests for `dependencytrack_team_permissions` resource using API key authentication
* tests: Added acceptance tests for username/password authentication on `dependencytrack_config_property` resource and data source
* tests: Added acceptance tests for username/password authentication on `dependencytrack_managed_user` resource and data source
* tests: Added dedicated pre-check functions (`testAccPreCheckAPIKey`, `testAccPreCheckUsernamePassword`) to skip tests based on available credentials
* tests: Added helper functions to explicitly test each authentication method independently
