# Terraform Provider for OWASP Dependency-Track

This [Terraform](https://www.terraform.io) provider is built using the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) and allows you to manage [OWASP Dependency-Track](https://dependencytrack.org/) resources.

This provider includes:

- Resources and data sources for managing Dependency-Track entities (`internal/provider/`)
- Examples (`examples/`) and generated documentation (`docs/`)
- Acceptance tests for resources and data sources

## Features

- **Team Management**: Create, read, update, and delete teams in Dependency-Track
- **User Management**: Create, read, update, and delete managed users in Dependency-Track
- **Example Resources**: Template resources, data sources, functions, and ephemeral resources for reference

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

## Using the provider

```hcl
terraform {
  required_providers {
    dependencytrack = {
      source = "ottramst/dependencytrack"
    }
  }
}

provider "dependencytrack" {
  endpoint = "https://dtrack.example.com"
  api_key  = var.dtrack_api_key
}

# Create a team
resource "dependencytrack_team" "security" {
  name = "Security Team"
}

# Query an existing team
data "dependencytrack_team" "existing" {
  uuid = "00000000-0000-0000-0000-000000000000"
}

# Create a user
resource "dependencytrack_user" "john" {
  username         = "johndoe"
  fullname         = "John Doe"
  email            = "john.doe@example.com"
  new_password     = "SecureP@ssw0rd123"
  confirm_password = "SecureP@ssw0rd123"
  type             = "managed"
}
```

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
