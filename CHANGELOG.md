## v0.5.0

FEATURES:

* **New Resource:** `dependencytrack_repository` - Manage package repositories in Dependency-Track
* **New Resource:** `dependencytrack_oidc_group` - Manage OpenID Connect groups
* **New Resource:** `dependencytrack_oidc_group_mapping` - Map OpenID Connect groups to teams
* **New Resource:** `dependencytrack_ldap_mapping` - Map LDAP distinguished names to teams
* **New Resource:** `dependencytrack_tag` - Manage portfolio tags
* **New Resource:** `dependencytrack_policy_tag` - Scope a policy to projects by tag
* **New Resource:** `dependencytrack_notification_rule_tag` - Scope a notification rule to projects by tag
* **New Resource:** `dependencytrack_license` - Manage custom licenses
* **New Resource:** `dependencytrack_license_group` - Manage license groups
* **New Resource:** `dependencytrack_license_group_license` - Manage license-group membership
* **New Resource:** `dependencytrack_project_property` - Manage project properties
* **New Data Source:** `dependencytrack_repositories` - List package repositories
* **New Data Source:** `dependencytrack_oidc_group` - Look up an OpenID Connect group
* **New Data Source:** `dependencytrack_tags` - List portfolio tags
* **New Data Source:** `dependencytrack_license` - Look up a license by license ID or UUID
* **New Data Source:** `dependencytrack_licenses` - List all licenses
* **New Data Source:** `dependencytrack_license_group` - Look up a license group
* **New Data Source:** `dependencytrack_portfolio_metrics` - Read the latest portfolio-wide metrics snapshot
* **New Data Source:** `dependencytrack_project_metrics` - Read the latest metrics snapshot for a project
* **New Data Source:** `dependencytrack_project_violations` - List a project's policy violations
* **New Data Source:** `dependencytrack_project_findings` - List a project's vulnerability findings

ENHANCEMENTS:

* provider: Added support for Dependency-Track v5 (tested against 5.0.2) alongside v4 (tested against 4.14.2). The server's major version is detected automatically at configure time by probing the unauthenticated `GET /api/version` endpoint; there is no version attribute to set, and configuration fails with an actionable error if the version cannot be detected
* provider: List and read operations paginate correctly on Dependency-Track v5, which caps list page size at 100, including client-go list methods that do not report a total count
* resource/notification_publisher: `publisher_class` accepts a fully qualified Java class name on v4 and a publisher extension name (e.g. `webhook`, `email`) on v5, and warns when the value's shape does not match the detected server
* resource/notification_rule: `publisher_config` no longer drifts on Dependency-Track v5, which stores it as JSONB and may re-serialize it or fill in publisher defaults; semantically equal JSON is treated as unchanged (consistent behavior on Dependency-Track >= 4.14)
* resource/project: The deprecated `author` field is preserved from configuration in state on v5, where the API accepts it on write but never returns it on read
* resource/project_property: Emits a warning when the `ENCRYPTEDSTRING` type is used against v5, which does not support it
* resource/repository: `password` is treated as write-only and preserved from state (Dependency-Track never returns it); on v5 it is the name of an existing secret, on v4 it is the literal password. `authentication_required` is always sent with a concrete value
* resource/tag: Tag names are normalized to lowercase to match Dependency-Track's storage, avoiding drift from mixed-case input

NOTES:

* deps: Bumped `github.com/DependencyTrack/client-go` to v0.19.0, `terraform-plugin-framework` to v1.19.0, `terraform-plugin-go` to v0.31.0, and `terraform-plugin-testing` to v1.16.0; the Go directive is now 1.25.8
* ci: The acceptance test matrix now runs against Dependency-Track 4.14.2 and 5.0.2 (both required) across Terraform 1.9, 1.12, and 1.15
* test: Added a `docker-compose.v5.yml` stack (Dependency-Track 5.0.2 on PostgreSQL 18) for local v5 testing; `scripts/init_dtrack.go` now polls `/api/version` for readiness, since v5 moved its health endpoints to a separate management port
* No breaking changes; no state migrations are required to adopt this release

## v0.4.1

BUG FIXES:

* resource/notification_rule: Fixed "Provider produced inconsistent result after apply" error when setting `publisher_config`. The API's PUT (create) endpoint ignores `publisher_config`, so the provider now includes it in the follow-up POST update and preserves the configured value in state

ENHANCEMENTS:

* tests: Added acceptance test `TestAccNotificationRuleResource_WithPublisherConfig` to verify `publisher_config` is preserved through create, import, and update operations

## v0.4.0

FEATURES:

* **New Data Source:** `dependencytrack_notification_publisher` - Look up notification publishers by UUID or name

ENHANCEMENTS:

* docs: Added documentation with examples for the `dependencytrack_notification_publisher` data source
* tests: Added acceptance tests for `dependencytrack_notification_publisher` data source (by UUID, by name, and both)

## v0.3.0

FEATURES:

* **New Resource:** `dependencytrack_project_policy` - Manage the assignment of policies to projects in Dependency-Track
* **New Resource:** `dependencytrack_notification_publisher` - Manage notification publishers in Dependency-Track
* **New Resource:** `dependencytrack_notification_rule` - Manage notification rules in Dependency-Track
* **New Resource:** `dependencytrack_notification_rule_project` - Associate projects with notification rules
* **New Resource:** `dependencytrack_notification_rule_team` - Associate teams with notification rules

ENHANCEMENTS:

* docs: Added documentation with examples for the `dependencytrack_notification_publisher` resource
* docs: Added documentation with examples for the `dependencytrack_notification_rule_project` resource
* docs: Added documentation with examples for the `dependencytrack_notification_rule_team` resource
* docs: Added documentation with examples for the `dependencytrack_notification_rule` resource
* docs: Added documentation with examples for the `dependencytrack_project_policy` resource
* resource/notification_publisher: Supports all notification publisher types (Webhook, Email, Console, etc.)
* resource/notification_publisher: Supports full CRUD operations for notification publishers
* resource/notification_publisher: Supports import using UUID
* resource/notification_rule: Supports all notification scopes (PORTFOLIO, SYSTEM) and levels (INFORMATIONAL, WARNING, ERROR)
* resource/notification_rule: Supports full CRUD operations for notification rules
* resource/notification_rule: Supports import using UUID
* resource/notification_rule_project: Supports associating projects with PORTFOLIO-scoped notification rules
* resource/notification_rule_project: Supports import using the format `rule_uuid/project_uuid`
* resource/notification_rule_team: Supports associating teams with EMAIL-publisher notification rules
* resource/notification_rule_team: Supports import using the format `rule_uuid/team_uuid`
* resource/project_policy: Supports full CRUD operations for project policy assignments
* resource/project_policy: Supports import using the format `policy_uuid/project_uuid`
* tests: Added acceptance tests for `dependencytrack_notification_publisher` resource using API key authentication
* tests: Added acceptance tests for `dependencytrack_notification_rule` resource using API key authentication
* tests: Added acceptance tests for `dependencytrack_project_policy` resource using API key authentication

## v0.2.2

BUG FIXES:

* resource/config_property: Fixed "Provider produced inconsistent result after apply" error when managing `ENCRYPTEDSTRING` type properties. The provider now correctly handles the API's `HiddenDecryptedPropertyPlaceholder` response and preserves the configured value in state

ENHANCEMENTS:

* tests: Added acceptance test `TestAccConfigPropertyResource_EncryptedString_APIKey` to verify encrypted config property handling
* docs: Removed read-only fields `include_children` and `global` from `dependencytrack_policy` resource example

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
