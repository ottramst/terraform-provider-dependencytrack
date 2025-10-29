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

func testAccConfigPropertyResourceConfigWithAPIKey(groupName, name, value string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_config_property" "test" {
  group_name = %[1]q
  name       = %[2]q
  value      = %[3]q
}
`, groupName, name, value)
}
