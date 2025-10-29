package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccManagedUserResource_APIKey tests the managed_user resource with API key authentication.
func TestAccManagedUserResource_APIKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccManagedUserResourceConfigWithAPIKey("apikey_testuser", "API Key Test User", "apikey@example.com", "P@ssw0rd123"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("apikey_testuser"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("fullname"),
						knownvalue.StringExact("API Key Test User"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact("apikey@example.com"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "dependencytrack_managed_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update and Read testing
			{
				Config: testAccManagedUserResourceConfigWithAPIKey("apikey_testuser", "Updated API Key Test User", "apikey_updated@example.com", "P@ssw0rd456"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("fullname"),
						knownvalue.StringExact("Updated API Key Test User"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact("apikey_updated@example.com"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccManagedUserResourceConfigWithAPIKey(username, fullname, email, password string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_managed_user" "test" {
  username = %[1]q
  fullname = %[2]q
  email    = %[3]q
  password = %[4]q
}
`, username, fullname, email, password)
}

// TestAccManagedUserResource_UsernamePassword tests the managed_user resource with username/password authentication.
func TestAccManagedUserResource_UsernamePassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckUsernamePassword(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccManagedUserResourceConfigWithUsernamePassword("userpass_testuser", "Username Password Test User", "userpass@example.com", "P@ssw0rd123"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("userpass_testuser"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("fullname"),
						knownvalue.StringExact("Username Password Test User"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact("userpass@example.com"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "dependencytrack_managed_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update and Read testing
			{
				Config: testAccManagedUserResourceConfigWithUsernamePassword("userpass_testuser", "Updated Username Password Test User", "userpass_updated@example.com", "P@ssw0rd456"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("fullname"),
						knownvalue.StringExact("Updated Username Password Test User"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact("userpass_updated@example.com"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccManagedUserResourceConfigWithUsernamePassword(username, fullname, email, password string) string {
	return testAccProviderConfigWithUsernamePassword() + fmt.Sprintf(`
resource "dependencytrack_managed_user" "test" {
  username = %[1]q
  fullname = %[2]q
  email    = %[3]q
  password = %[4]q
}
`, username, fullname, email, password)
}
