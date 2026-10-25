# Contributing

Thanks for your interest in improving the Dependency-Track Terraform provider. Bug reports, feature
requests and pull requests are all welcome. By participating you agree to follow the
[Code of Conduct](.github/CODE_OF_CONDUCT.md).

## Reporting issues

Use the [issue templates](https://github.com/ottramst/terraform-provider-dependencytrack/issues/new/choose).
Always include the provider, Terraform and Dependency-Track versions. Report security vulnerabilities
privately, as described in [SECURITY.md](SECURITY.md).

## Development setup

Requirements: [Go](https://go.dev/doc/install) (the version in `go.mod`), [Terraform](https://developer.hashicorp.com/terraform/install),
Docker, and [golangci-lint](https://golangci-lint.run/) (the version pinned in `.github/workflows/lint.yml`).

```shell
make build      # build the provider
make test       # unit tests
make lint       # golangci-lint
make generate   # regenerate docs/ from the schemas and examples/
```

`make generate` must produce no diff in CI, so commit the regenerated `docs/` with every schema or
example change. Never edit `docs/` by hand: change the schema descriptions or `examples/` instead.

## Acceptance tests

Acceptance tests run against a real Dependency-Track instance. Docker Compose stacks are provided for
both major versions; `DT_VERSION` selects the image tag and defaults to the newest version of each line.

```shell
docker compose up -d                                   # Dependency-Track v4
# DT_VERSION=5.2.1 docker compose -f docker-compose.v5.yml up -d   # Dependency-Track v5

export DEPENDENCYTRACK_API_KEY="$(go run scripts/init_dtrack.go)"
export DEPENDENCYTRACK_ENDPOINT="http://localhost:8081"
export DEPENDENCYTRACK_USERNAME="admin"
export DEPENDENCYTRACK_PASSWORD="admin123"

make testacc
# or a single test:
TF_ACC=1 go test -v -run TestAccTeamResource ./internal/provider/ -timeout 120m

docker compose down -v
```

Behavior that differs between Dependency-Track versions must be covered for each supported version.
Gate version-specific features with `requireV5` / `requireAtLeast` in the provider, and skip tests with
`testAccSkipUnlessV4`, `testAccSkipUnlessV5` or `testAccSkipUnlessAtLeast`.

## Supported Dependency-Track versions

CI tests the latest patch of every non-end-of-life Dependency-Track release line (see the
[support policy](README.md#support-policy)). When upstream ships a new release, bump these together:

1. The matrix in `.github/workflows/test.yml`
2. The `DT_VERSION` defaults in `docker-compose.yml` / `docker-compose.v5.yml`
3. The tested versions in `README.md`, the provider description in `internal/provider/provider.go`, and `CLAUDE.md`

When a release line reaches end-of-life, remove it from the matrix; it then gets best-effort support.

The same applies to Terraform: the `terraform-version` matrix in `.github/workflows/test.yml` covers the two
newest Terraform minor releases (the ones HashiCorp supports). When a new minor ships, add it, drop the oldest,
and update the versions in `README.md` and `CLAUDE.md`.

## Pull requests

- Keep each pull request focused on one change, and add or update acceptance tests for it.
- Add an entry to `CHANGELOG.md` under `## Unreleased` (`FEATURES`, `BUG FIXES` or `NOTES`).
- CI must pass: the `CI gate` check covers docs generation, lint, build, unit tests and the acceptance
  test matrix.

## Releases

Maintainers release by pushing a `vX.Y.Z` tag. The release workflow builds and signs the artifacts with
GoReleaser, attests their build provenance, and publishes a GitHub release that the Terraform Registry picks up.
Before tagging, move the `## Unreleased` entries in `CHANGELOG.md` under the new version.
