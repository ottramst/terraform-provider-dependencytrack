package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccConfigPropertyDataSource_APIKey tests the config_property data source with API key authentication.
func TestAccConfigPropertyDataSource_APIKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a config property resource first, then read it via data source
			{
				Config: testAccConfigPropertyDataSourceConfigWithAPIKey,
				ConfigStateChecks: []statecheck.StateCheck{
					// Check the resource was created/adopted
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
					// Check the data source can read it
					statecheck.ExpectKnownValue(
						"data.dependencytrack_config_property.test",
						tfjsonpath.New("group_name"),
						knownvalue.StringExact("general"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_config_property.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("base.url"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_config_property.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("https://apikey-datasource.example.com"),
					),
				},
			},
		},
	})
}

var testAccConfigPropertyDataSourceConfigWithAPIKey = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_config_property" "test" {
  group_name = "general"
  name       = "base.url"
  value      = "https://apikey-datasource.example.com"
}

data "dependencytrack_config_property" "test" {
  group_name = dependencytrack_config_property.test.group_name
  name       = dependencytrack_config_property.test.name
}
`
