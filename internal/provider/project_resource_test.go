package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig("Test Project", "1.0.0", "Test Description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Project"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact("1.0.0"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test Description"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("active"),
						knownvalue.Bool(true),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccProjectResourceConfig("Test Project", "1.0.1", "Updated Description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact("1.0.1"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated Description"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfig(name, version, description string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_project" "test" {
  name        = %[1]q
  version     = %[2]q
  description = %[3]q
}
`, name, version, description)
}

func TestAccProjectResource_WithOptionalFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all optional fields
			{
				Config: testAccProjectResourceConfigComplete(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Complete Project"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact("2.0.0"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("com.example"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("publisher"),
						knownvalue.StringExact("Example Publisher"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("author"),
						knownvalue.StringExact("Example Author"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("classifier"),
						knownvalue.StringExact("APPLICATION"),
					),
				},
			},
		},
	})
}

func testAccProjectResourceConfigComplete() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_project" "test" {
  name        = "Complete Project"
  version     = "2.0.0"
  description = "A complete test project"
  group       = "com.example"
  publisher   = "Example Publisher"
  author      = "Example Author"
  classifier  = "APPLICATION"
  active      = true
}
`
}
