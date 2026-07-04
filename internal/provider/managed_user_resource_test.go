package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccManagedUserResource tests the managed_user resource.
func TestAccManagedUserResource(t *testing.T) {
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

// TestAccManagedUserResource_BooleanToggle tests that suspended,
// force_password_change, and non_expiry_password can be toggled from
// true back to false on update. This guards against the fields being
// dropped from the update request body (e.g. via json omitempty),
// which would leave them true on the server.
func TestAccManagedUserResource_BooleanToggle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all boolean attributes set to true
			{
				Config: testAccManagedUserResourceConfigWithBooleans("booltoggle_testuser", "Bool Toggle Test User", "booltoggle@example.com", "P@ssw0rd123", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("suspended"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("force_password_change"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("non_expiry_password"),
						knownvalue.Bool(true),
					),
				},
			},
			// Update all boolean attributes back to false
			{
				Config: testAccManagedUserResourceConfigWithBooleans("booltoggle_testuser", "Bool Toggle Test User", "booltoggle@example.com", "P@ssw0rd123", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("suspended"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("force_password_change"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user.test",
						tfjsonpath.New("non_expiry_password"),
						knownvalue.Bool(false),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccManagedUserResourceConfigWithBooleans(username, fullname, email, password string, flag bool) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_managed_user" "test" {
  username              = %[1]q
  fullname              = %[2]q
  email                 = %[3]q
  password              = %[4]q
  suspended             = %[5]t
  force_password_change = %[5]t
  non_expiry_password   = %[5]t
}
`, username, fullname, email, password, flag)
}
