# Terraform Provider for OWASP Dependency-Track

This [Terraform](https://www.terraform.io) provider is built using the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) and allows you to manage [OWASP Dependency-Track](https://dependencytrack.org/) resources.

This provider includes:

- Resources and data sources for managing Dependency-Track entities (`internal/provider/`)
- Examples (`examples/`) and generated documentation (`docs/`)
- Acceptance tests for resources and data sources

## Features

- **Teams & API keys**: Manage teams, team permissions, and team API keys
- **Users & access control**: Manage managed users, user permissions, user team memberships, OIDC groups and mappings, and LDAP mappings
- **Projects**: Manage projects, project properties, ACL mappings, and project-to-policy assignments
- **Policies**: Manage policies with conditions and scope them to projects by tag
- **Notifications**: Manage notification publishers and rules, and scope rules by project, team, or tag
- **Licenses**: Manage custom licenses, license groups, and license-group membership
- **Repositories, tags & configuration**: Manage package repositories, portfolio tags, and configuration properties
- **Metrics & findings**: Read portfolio and project metrics, project findings, and policy violations via data sources
- **Dependency-Track v4 and v5**: Runtime version auto-detection with version-aware behavior (see [compatibility](#dependency-track-version-compatibility) below)

For the complete list of resources and data sources with detailed documentation, see the [docs/](docs/) directory.

## Dependency-Track version compatibility

The provider supports both current Dependency-Track major release lines. Acceptance tests run in CI against **Dependency-Track 4.14.2** and **5.0.2** (both legs are required to pass) across Terraform 1.9, 1.12, and 1.15.

At configure time the provider makes an unauthenticated `GET {endpoint}/api/version` request to detect the server's major version and adapt version-dependent behavior automatically. There is no version attribute to set. If that probe fails — an unreachable endpoint, a proxy in the way, or an unparseable version string — provider configuration fails with an actionable error rather than silently guessing a version.

A handful of attributes behave differently depending on the detected version:

| Area | Dependency-Track v4 | Dependency-Track v5 |
| --- | --- | --- |
| `notification_publisher.publisher_class` | Fully qualified Java class name (e.g. `org.dependencytrack.notification.publisher.WebhookPublisher`) | Extension name (e.g. `webhook`, `email`). The provider warns when the value's shape does not match the detected server. |
| `notification_rule.publisher_config` | Preserved as configured | Stored as JSONB and may be re-serialized or have publisher defaults filled in; the provider treats semantically equal JSON as unchanged to avoid perpetual drift |
| `project.author` | Returned on read | Deprecated: accepted on write but never returned; the provider preserves the configured value in state |
| `project_property.type = ENCRYPTEDSTRING` | Supported | Rejected by the server; the provider emits a warning |
| `config_property.type = ENCRYPTEDSTRING` | Exists | v5 exposes no `ENCRYPTEDSTRING` config properties |
| `repository.password` | Literal password (write-only; never read back, preserved from state) | Name of an existing Dependency-Track secret (write-only; never read back, preserved from state) |
| Team / user permissions | Full v4 permission set (e.g. `VIEW_BADGES`) | Permission names are passed through verbatim; the valid set is defined by the server and can differ between major versions |

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 (protocol 6; CI covers 1.9, 1.12, and 1.15)
- [Go](https://golang.org/doc/install) >= 1.25 (only required to build the provider from source)
- [OWASP Dependency-Track](https://dependencytrack.org/) 4.14.x or 5.0.x (see [compatibility](#dependency-track-version-compatibility))

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the Provider

For usage examples and comprehensive documentation, please refer to the:
- [Provider documentation](docs/index.md) for configuration details
- [Resource documentation](docs/resources/) for available resources
- [Data source documentation](docs/data-sources/) for available data sources
- [Example configurations](examples/) for complete working examples

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

### Running Acceptance Tests

Acceptance tests require a running Dependency-Track instance. A local stack is provided for both supported major versions: `docker-compose.yml` (v4) and `docker-compose.v5.yml` (v5).

```shell
# Start a local Dependency-Track (use docker-compose.v5.yml for v5)
docker compose up -d

# Initialize it and capture an API key (waits for /api/version to respond)
export DEPENDENCYTRACK_API_KEY="$(go run scripts/init_dtrack.go)"
export DEPENDENCYTRACK_ENDPOINT="http://localhost:8081"
export DEPENDENCYTRACK_USERNAME="admin"
export DEPENDENCYTRACK_PASSWORD="admin123"
```

Both API key and username/password variables should be set, since different tests exercise different authentication methods. Then run the acceptance tests:

```shell
make testacc
```

*Note:* Acceptance tests create real resources in your Dependency-Track instance. Make sure to use a test instance and not a production environment.
