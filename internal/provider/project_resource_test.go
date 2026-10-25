package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
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

// TestAccProjectResource_Parent covers setting, changing and removing a
// project's parent. Removing parent_uuid must clear the parent on the server:
// v5 documents that an update without a parent clears it (#7312), while v4
// documents it as leaving the parent unchanged.
func TestAccProjectResource_Parent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceConfigParent("dependencytrack_project.parent_a.id"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(
						"dependencytrack_project.test", tfjsonpath.New("parent_uuid"),
						"dependencytrack_project.parent_a", tfjsonpath.New("id"),
						compare.ValuesSame(),
					),
				},
			},
			{
				Config: testAccProjectResourceConfigParent("dependencytrack_project.parent_b.id"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs(
						"dependencytrack_project.test", tfjsonpath.New("parent_uuid"),
						"dependencytrack_project.parent_b", tfjsonpath.New("id"),
						compare.ValuesSame(),
					),
				},
			},
			{
				Config: testAccProjectResourceConfigParent("null"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("parent_uuid"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

func testAccProjectResourceConfigParent(parentRef string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_project" "parent_a" {
  name    = "Parent Project A"
  version = "1.0.0"
}

resource "dependencytrack_project" "parent_b" {
  name    = "Parent Project B"
  version = "1.0.0"
}

resource "dependencytrack_project" "test" {
  name        = "Child Project"
  version     = "1.0.0"
  parent_uuid = %[1]s
}
`, parentRef)
}
