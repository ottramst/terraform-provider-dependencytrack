# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terraform provider for [OWASP Dependency-Track](https://dependencytrack.org/) built with the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework). Uses `github.com/DependencyTrack/client-go` as the primary API client. All provider code lives in `internal/provider/`.

The provider supports both Dependency-Track v4 (tested against 4.14.x) and v5 (tested against 5.0.x). The server's major version is detected at runtime (see Architecture) and behavior is adapted accordingly; there is no provider version attribute.

## Common Commands

```bash
make default          # fmt, lint, install, generate (full dev cycle)
make build            # Build binary
make lint             # Run golangci-lint
make fmt              # Format Go code with gofmt
make generate         # Generate provider documentation (tfplugindocs) + terraform fmt examples
make test             # Unit tests (120s timeout, parallel=10)
make testacc          # Acceptance tests (requires running Dependency-Track instance)
```

Run a single test:
```bash
TF_ACC=1 go test -v -run TestAccTeamResource ./internal/provider/ -timeout 120m
```

## Running Acceptance Tests Locally

Acceptance tests require a running Dependency-Track instance. Local stacks are provided for both major versions:

```bash
# 1. Start the local stack (API server, frontend, PostgreSQL).
#    Use docker-compose.v5.yml to test against Dependency-Track v5 instead.
docker compose up -d                       # v4 (docker-compose.yml)
# docker compose -f docker-compose.v5.yml up -d   # v5

# 2. Initialize DependencyTrack and get an API key.
#    Waits for GET /api/version to respond, changes the default admin password
#    from "admin" to "admin123", and generates an API key for the Administrators
#    team. First run takes several minutes while DependencyTrack initializes.
export DEPENDENCYTRACK_API_KEY="$(go run scripts/init_dtrack.go)"

# 3. Set the remaining environment variables
export DEPENDENCYTRACK_ENDPOINT="http://localhost:8081"
export DEPENDENCYTRACK_USERNAME="admin"
export DEPENDENCYTRACK_PASSWORD="admin123"

# 4. Run all acceptance tests
make testacc

# Or run a specific test
TF_ACC=1 go test -v -run TestAccTeamResource ./internal/provider/ -timeout 120m

# 5. Tear down when done
docker compose down -v
```

The init script (`scripts/init_dtrack.go`) polls `GET /api/version` with exponential backoff (up to 20 retries) rather than the `/health` endpoints, because v5 moved health checks to a separate management port (9000) that is not exposed by the compose stacks. Both API key and username/password env vars should be set since different tests exercise different auth methods. Optionally set `DEPENDENCYTRACK_SERVER_VERSION` to skip the version probe in tests (see Test Patterns).

## Architecture

**Provider entry point:** `main.go` → `internal/provider/provider.go`

The provider authenticates via API key OR username/password (mutually exclusive). At `Configure()` it probes the unauthenticated `GET /api/version` endpoint (via client-go's `About.Get`) to detect the server version; if the probe fails or the version is unparseable, configuration fails with an actionable diagnostic (no silent fallback). Provider config is stored in the `Data` struct (`provider.go`), which carries the `dtrack.Client`, endpoint, auth credentials, the parsed `ServerVersion`, and a shared `*apiClient`. This struct is passed to every resource/data source via its `Configure()` method (each holds a `*Data`, not a bare client).

**Version detection (`version.go`):** `ServerVersion` holds `Major`/`Minor` parsed from the version string. Use `Data.IsV5()` (or `ServerVersion.IsV5()` / `ServerVersion.AtLeast(major, minor)`) to branch on version. Version-dependent behavior currently lives in `provider.go`, `notification_publisher_resource.go`, `project_resource.go`, and `project_property_resource.go`.

**Shared HTTP client (`apiclient.go`):** `apiClient` (reachable via `Data.API()`) is a small helper for Dependency-Track endpoints not covered by client-go's typed methods. It centralizes base-URL handling, auth headers, JSON encoding, error classification (`isNotFound`/`isForbidden`), and pagination. Only the `notification_*` resources/data source and `user_team_membership` use it; everything else uses client-go via `Data.Client`. Two pagination helpers request fixed pages of 100 (v5 caps list `pageSize` at 100) and stop on a short page rather than trusting a total count: `apiGetAllPages` (raw `apiClient`) and `fetchAllPages` (client-go list methods, several of which never populate `TotalCount`).

**Resource pattern** (all files in `internal/provider/`):
- Each resource has a struct holding `*Data` and a model struct with `tfsdk` tags (e.g. `TeamResourceModel`)
- Implements `resource.Resource` and usually `resource.ResourceWithImportState`
- Methods: `Metadata()`, `Schema()`, `Configure()`, `Create()`, `Read()`, `Update()`, `Delete()`, `ImportState()`
- Type name format: `dependencytrack_<name>` (set in `Metadata()`)
- Errors use `resp.Diagnostics.AddError()` with title + detail; version mismatches use `AddWarning`/`AddAttributeWarning`

**Composite IDs (`helpers.go`):** Resources that reference multiple entities use slash-delimited IDs, parsed via `parseCompositeID()` for two parts (e.g. `acl_mapping`, `project_policy`, `notification_rule_project` in `uuid1/uuid2` form) and `parseCompositeID3()` for three parts (e.g. `project_property`, `project_uuid/group/name`). `helpers.go` also has `jsonStringsEquivalent`/`canonicalJSONString` for JSON attributes such as `notification_rule.publisher_config` that the server may re-serialize.

## Test Patterns

Acceptance tests require a real Dependency-Track instance. Tests follow this structure:
- `TestAcc<Resource>Resource(t)` — CRUD + import in sequential steps
- `testAcc<Resource>ResourceConfig(...)` — generates HCL, prefixed with provider config from `testAccProviderConfigWithAPIKey()` or `testAccProviderConfigWithUsernamePassword()`
- PreCheck functions (`testAccPreCheckAPIKey`, `testAccPreCheckUsernamePassword`) validate env vars
- Uses `statecheck.ExpectKnownValue` with `tfjsonpath` for assertions
- Provider factory: `testAccProtoV6ProviderFactories`

**Version-aware test helpers** (`provider_test.go`):
- `testAccServerVersion(t)` — resolves the server version once per process, preferring `DEPENDENCYTRACK_SERVER_VERSION` and otherwise querying `{endpoint}/api/version`
- `testAccSkipUnlessV4(t)` / `testAccSkipUnlessV5(t)` — skip a test unless the server is the matching major version
- `testAccPublisherClass(t)` / `testAccEmailPublisherClass(t)` — return a notification `publisher_class` valid for the server under test (FQCN on v4, extension name on v5)

## CI

GitHub Actions workflows in `.github/workflows/` (`main.yml` orchestrates `generate`, `lint`, `build`, `test`):
- Acceptance tests run a matrix of Dependency-Track {4.14.2, 5.0.2} (both required) × Terraform {1.9.\*, 1.12.\*, 1.15.\*}
- All `setup-go` steps use `go-version-file: go.mod`
- `make generate` must produce no git diff
- golangci-lint enforced with zero tolerance (`max-issues-per-linter: 0`)
