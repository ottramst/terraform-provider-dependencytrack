package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectPropertyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectPropertyResourceConfig("acme", "color", "blue", "STRING"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_project_property.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("acme"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project_property.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("color"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project_property.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("blue"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project_property.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("STRING"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_project_property.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update (value only) and Read testing
			{
				Config: testAccProjectPropertyResourceConfig("acme", "color", "red", "STRING"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_project_property.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("red"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectPropertyResourceConfig(group, name, value, propertyType string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_project" "test" {
  name    = "Test Project Property Project"
  version = "1.0.0"
}

resource "dependencytrack_project_property" "test" {
  project     = dependencytrack_project.test.id
  group       = %[1]q
  name        = %[2]q
  value       = %[3]q
  type        = %[4]q
  description = "a project property"
}
`, group, name, value, propertyType)
}
