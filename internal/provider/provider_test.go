package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
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
// This explicitly tests API key authentication regardless of which environment variables are set.
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
// This explicitly tests username/password authentication regardless of which environment variables are set.
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

// testAccPreCheckUsernamePassword checks that username/password authentication is available for acceptance tests.
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
