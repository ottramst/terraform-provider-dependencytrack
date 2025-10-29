package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccACLMappingResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccACLMappingResourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_acl_mapping.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_acl_mapping.test",
						tfjsonpath.New("project"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_acl_mapping.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccACLMappingResourceConfig() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test ACL Team"
}

resource "dependencytrack_project" "test" {
  name    = "Test ACL Project"
  version = "1.0.0"
}

resource "dependencytrack_acl_mapping" "test" {
  team    = dependencytrack_team.test.id
  project = dependencytrack_project.test.id
}
`
}

func TestAccACLMappingResource_Multiple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple ACL mappings
			{
				Config: testAccACLMappingResourceConfigMultiple(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_acl_mapping.test1",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_acl_mapping.test2",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccACLMappingResourceConfigMultiple() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test ACL Team Multi"
}

resource "dependencytrack_project" "test1" {
  name    = "Test ACL Project 1"
  version = "1.0.0"
}

resource "dependencytrack_project" "test2" {
  name    = "Test ACL Project 2"
  version = "1.0.0"
}

resource "dependencytrack_acl_mapping" "test1" {
  team    = dependencytrack_team.test.id
  project = dependencytrack_project.test1.id
}

resource "dependencytrack_acl_mapping" "test2" {
  team    = dependencytrack_team.test.id
  project = dependencytrack_project.test2.id
}
`
}
