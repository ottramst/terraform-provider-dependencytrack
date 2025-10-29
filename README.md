# Terraform Provider for OWASP Dependency-Track

This [Terraform](https://www.terraform.io) provider is built using the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) and allows you to manage [OWASP Dependency-Track](https://dependencytrack.org/) resources.

This provider includes:

- Resources and data sources for managing Dependency-Track entities (`internal/provider/`)
- Examples (`examples/`) and generated documentation (`docs/`)
- Acceptance tests for resources and data sources

## Features

- **Team Management**: Manage teams, team permissions, and team API keys
- **User Management**: Manage managed users, user permissions, and user team memberships
- **Project Management**: Manage projects and project ACL mappings
- **Policy Management**: Manage policies with conditions
- **Configuration**: Manage Dependency-Track configuration properties

For detailed documentation on all resources and data sources, see the [docs/](docs/) directory.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24

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

In order to run the full suite of acceptance tests, you need to set the following environment variables:

```shell
export DEPENDENCYTRACK_ENDPOINT="https://dtrack.example.com"
export DEPENDENCYTRACK_API_KEY="your-api-key-here"
```

Then run the acceptance tests:

```shell
make testacc
```

*Note:* Acceptance tests create real resources in your Dependency-Track instance. Make sure to use a test instance and not a production environment.
