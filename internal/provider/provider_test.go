package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"dependencytrack": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the dependencytrack provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){ // nolint:unused
	"dependencytrack": providerserver.NewProtocol6WithError(New("test")()),
	"echo":            echoprovider.NewProviderServer(),
}

// testAccProviderConfigWithAPIKey returns the provider configuration block using API key authentication.
// This is the standard authentication used by all acceptance tests by convention;
// only TestAccProviderAuth_UsernamePassword uses username/password instead.
func testAccProviderConfigWithAPIKey() string {
	endpoint := os.Getenv("DEPENDENCYTRACK_ENDPOINT")
	apiKey := os.Getenv("DEPENDENCYTRACK_API_KEY")

	return `
provider "dependencytrack" {
  endpoint = "` + endpoint + `"
  api_key  = "` + apiKey + `"
}
`
}

// testAccProviderConfigWithUsernamePassword returns the provider configuration block using username/password authentication.
// It is used only by TestAccProviderAuth_UsernamePassword, the sole test that
// exercises the username/password -> User.Login bearer-token Configure() path;
// all other acceptance tests authenticate with an API key by convention.
func testAccProviderConfigWithUsernamePassword() string {
	endpoint := os.Getenv("DEPENDENCYTRACK_ENDPOINT")
	username := os.Getenv("DEPENDENCYTRACK_USERNAME")
	password := os.Getenv("DEPENDENCYTRACK_PASSWORD")

	return `
provider "dependencytrack" {
  endpoint = "` + endpoint + `"
  username = "` + username + `"
  password = "` + password + `"
}
`
}

// testAccPreCheckAPIKey checks that API key authentication is available for acceptance tests.
func testAccPreCheckAPIKey(t *testing.T) {
	if v := os.Getenv("DEPENDENCYTRACK_ENDPOINT"); v == "" {
		t.Skip("DEPENDENCYTRACK_ENDPOINT must be set for acceptance tests")
	}
	if v := os.Getenv("DEPENDENCYTRACK_API_KEY"); v == "" {
		t.Skip("DEPENDENCYTRACK_API_KEY must be set for API key authentication tests")
	}
}

// testAccPreCheckUsernamePassword checks that username/password authentication is available.
// It gates TestAccProviderAuth_UsernamePassword, the only test that authenticates
// with username/password rather than an API key.
func testAccPreCheckUsernamePassword(t *testing.T) {
	if v := os.Getenv("DEPENDENCYTRACK_ENDPOINT"); v == "" {
		t.Skip("DEPENDENCYTRACK_ENDPOINT must be set for acceptance tests")
	}
	if v := os.Getenv("DEPENDENCYTRACK_USERNAME"); v == "" {
		t.Skip("DEPENDENCYTRACK_USERNAME must be set for username/password authentication tests")
	}
	if v := os.Getenv("DEPENDENCYTRACK_PASSWORD"); v == "" {
		t.Skip("DEPENDENCYTRACK_PASSWORD must be set for username/password authentication tests")
	}
}

// TestAccProviderAuth_UsernamePassword is the sole end-to-end coverage of the
// provider's username/password authentication path. Configuring the provider
// with username/password (rather than an api_key) forces Configure() to call
// User.Login, exchanging the credentials for a bearer token used on every
// subsequent request. All other acceptance tests authenticate with an API key
// by convention; this test alone verifies the login path still works. It uses a
// minimal read-only config (looking up the built-in "Administrators" team by
// name) so it only exercises authentication, not any particular resource.
func TestAccProviderAuth_UsernamePassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckUsernamePassword(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfigWithUsernamePassword() + `
data "dependencytrack_team" "administrators" {
  name = "Administrators"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.administrators",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Administrators"),
					),
				},
			},
		},
	})
}

// State backing testAccServerVersion below.
var (
	testAccServerVersionOnce  sync.Once
	testAccServerVersionValue ServerVersion
	testAccServerVersionErr   error
)

// testAccServerVersion resolves the Dependency-Track server version under
// test, once per process. It prefers the DEPENDENCYTRACK_SERVER_VERSION
// environment variable when set (useful for CI matrices that already know
// the version), and otherwise queries {DEPENDENCYTRACK_ENDPOINT}/api/version
// directly. It calls t.Fatal if the version cannot be resolved.
//
// Like resource.Test, it skips the test when TF_ACC is unset, so helpers that
// consult the server version (e.g. testAccPublisherClass) can be called
// before resource.Test without breaking plain `go test` runs.
func testAccServerVersion(t *testing.T) ServerVersion {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	testAccServerVersionOnce.Do(func() {
		if v := os.Getenv("DEPENDENCYTRACK_SERVER_VERSION"); v != "" {
			testAccServerVersionValue, testAccServerVersionErr = parseServerVersion(v)
			return
		}

		endpoint := os.Getenv("DEPENDENCYTRACK_ENDPOINT")
		if endpoint == "" {
			testAccServerVersionErr = fmt.Errorf("DEPENDENCYTRACK_ENDPOINT must be set to resolve the server version")
			return
		}

		resp, err := http.Get(strings.TrimSuffix(endpoint, "/") + "/api/version")
		if err != nil {
			testAccServerVersionErr = fmt.Errorf("fetching %s/api/version: %w", endpoint, err)
			return
		}
		defer resp.Body.Close()

		var payload struct {
			Version string `json:"version"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			testAccServerVersionErr = fmt.Errorf("decoding %s/api/version response: %w", endpoint, err)
			return
		}

		testAccServerVersionValue, testAccServerVersionErr = parseServerVersion(payload.Version)
	})

	if testAccServerVersionErr != nil {
		t.Fatalf("unable to resolve Dependency-Track server version: %s", testAccServerVersionErr)
	}

	return testAccServerVersionValue
}

// testAccPublisherClass returns a webhook publisher_class value valid for the
// server under test: Dependency-Track v4 identifies notification publishers
// by fully qualified Java class name, while v5 identifies them by extension
// name (the v5.0.2 default publishers list "console", "email", "jira",
// "kafka", "mattermost", "msteams", "slack", "webex" and "webhook").
func testAccPublisherClass(t *testing.T) string {
	t.Helper()

	if testAccServerVersion(t).IsV5() {
		return "webhook"
	}
	return "org.dependencytrack.notification.publisher.WebhookPublisher"
}

// testAccEmailPublisherClass returns an email publisher_class value valid for
// the server under test. Notification rule team subscriptions require an
// email publisher on Dependency-Track v4 (other publishers are rejected with
// HTTP 406 "Team subscriptions are only possible on notification rules with
// EMAIL publisher").
func testAccEmailPublisherClass(t *testing.T) string {
	t.Helper()

	if testAccServerVersion(t).IsV5() {
		return "email"
	}
	return "org.dependencytrack.notification.publisher.SendMailPublisher"
}

// testAccSkipUnlessV4 skips the current test unless the server under test is
// running Dependency-Track 4.x.
func testAccSkipUnlessV4(t *testing.T) {
	t.Helper()

	if testAccServerVersion(t).IsV5() {
		t.Skip("test requires a Dependency-Track v4 server")
	}
}

// testAccSkipUnlessV5 skips the current test unless the server under test is
// running Dependency-Track 5.x or newer.
//
//nolint:unused // not yet called; v5-only acceptance tests land in a later task
func testAccSkipUnlessV5(t *testing.T) {
	t.Helper()

	if !testAccServerVersion(t).IsV5() {
		t.Skip("test requires a Dependency-Track v5 server")
	}
}
