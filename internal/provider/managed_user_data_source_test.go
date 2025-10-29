package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccManagedUserDataSource_APIKey tests the managed_user data source with API key authentication.
func TestAccManagedUserDataSource_APIKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a managed user resource first, then read it via data source
			{
				Config: testAccManagedUserDataSourceConfigWithAPIKey,
				ConfigStateChecks: []statecheck.StateCheck{
					// Check the resource was created
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("apikey_datasourcetest"),
					),
					// Check the data source can read it
					statecheck.ExpectKnownValue(
						"data.dependencytrack_managed_user.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("apikey_datasourcetest"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_managed_user.test",
						tfjsonpath.New("fullname"),
						knownvalue.StringExact("API Key DataSource Test User"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_managed_user.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact("apikey_dstest@example.com"),
					),
				},
			},
		},
	})
}

var testAccManagedUserDataSourceConfigWithAPIKey = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_managed_user" "test" {
  username = "apikey_datasourcetest"
  fullname = "API Key DataSource Test User"
  email    = "apikey_dstest@example.com"
  password = "TestP@ssw0rd123"
}

data "dependencytrack_managed_user" "test" {
  username = dependencytrack_managed_user.test.username
}
`

// TestAccManagedUserDataSource_UsernamePassword tests the managed_user data source with username/password authentication.
func TestAccManagedUserDataSource_UsernamePassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckUsernamePassword(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a managed user resource first, then read it via data source
			{
				Config: testAccManagedUserDataSourceConfigWithUsernamePassword,
				ConfigStateChecks: []statecheck.StateCheck{
					// Check the resource was created
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("userpass_datasourcetest"),
					),
					// Check the data source can read it
					statecheck.ExpectKnownValue(
						"data.dependencytrack_managed_user.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("userpass_datasourcetest"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_managed_user.test",
						tfjsonpath.New("fullname"),
						knownvalue.StringExact("Username Password DataSource Test User"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_managed_user.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact("userpass_dstest@example.com"),
					),
				},
			},
		},
	})
}

var testAccManagedUserDataSourceConfigWithUsernamePassword = testAccProviderConfigWithUsernamePassword() + `
resource "dependencytrack_managed_user" "test" {
  username = "userpass_datasourcetest"
  fullname = "Username Password DataSource Test User"
  email    = "userpass_dstest@example.com"
  password = "TestP@ssw0rd123"
}

data "dependencytrack_managed_user" "test" {
  username = dependencytrack_managed_user.test.username
}
`
