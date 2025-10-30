package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccConfigPropertyResource_APIKey tests the config_property resource with API key authentication.
func TestAccConfigPropertyResource_APIKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Adopt and Update testing
			{
				Config: testAccConfigPropertyResourceConfigWithAPIKey("general", "base.url", "https://apikey.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("group_name"),
						knownvalue.StringExact("general"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("base.url"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("https://apikey.example.com"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_config_property.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccConfigPropertyResourceConfigWithAPIKey("general", "base.url", "https://apikey-updated.example.com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("https://apikey-updated.example.com"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccConfigPropertyResource_EncryptedString_APIKey tests the config_property resource with an encrypted string property.
// This test ensures that encrypted properties (ENCRYPTEDSTRING type) are handled correctly,
// as the API returns "HiddenDecryptedPropertyPlaceholder" instead of the actual value.
func TestAccConfigPropertyResource_EncryptedString_APIKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Adopt and Update testing with encrypted property
			{
				Config: testAccConfigPropertyResourceConfigWithAPIKey("email", "smtp.password", "initial-password-123"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("group_name"),
						knownvalue.StringExact("email"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("smtp.password"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("initial-password-123"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("ENCRYPTEDSTRING"),
					),
				},
			},
			// ImportState testing
			// Note: We cannot verify the value field for encrypted properties because
			// the API returns "HiddenDecryptedPropertyPlaceholder" instead of the actual value
			{
				ResourceName:            "dependencytrack_config_property.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
			// Update testing - this is critical for encrypted properties
			// Previously this would fail with "Provider produced inconsistent result after apply"
			{
				Config: testAccConfigPropertyResourceConfigWithAPIKey("email", "smtp.password", "updated-password-456"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("updated-password-456"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("ENCRYPTEDSTRING"),
					),
				},
			},
			// Another update to ensure consistency is maintained
			{
				Config: testAccConfigPropertyResourceConfigWithAPIKey("email", "smtp.password", "final-password-789"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_config_property.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("final-password-789"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccConfigPropertyResourceConfigWithAPIKey(groupName, name, value string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_config_property" "test" {
  group_name = %[1]q
  name       = %[2]q
  value      = %[3]q
}
`, groupName, name, value)
}
