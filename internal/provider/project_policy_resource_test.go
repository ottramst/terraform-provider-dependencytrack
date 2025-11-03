package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectPolicyResourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_project_policy.test",
						tfjsonpath.New("policy"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project_policy.test",
						tfjsonpath.New("project"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_project_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectPolicyResourceConfig() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_project" "test" {
  name    = "Test Project Policy Project"
  version = "1.0.0"
}

resource "dependencytrack_policy" "test" {
  name            = "Test Project Policy"
  operator        = "ANY"
  violation_state = "INFO"
}

resource "dependencytrack_project_policy" "test" {
  policy  = dependencytrack_policy.test.id
  project = dependencytrack_project.test.id
}
`
}

func TestAccProjectPolicyResource_Multiple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple project policy assignments
			{
				Config: testAccProjectPolicyResourceConfigMultiple(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_project_policy.test1",
						tfjsonpath.New("policy"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_project_policy.test2",
						tfjsonpath.New("policy"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccProjectPolicyResourceConfigMultiple() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_project" "test1" {
  name    = "Test Project 1"
  version = "1.0.0"
}

resource "dependencytrack_project" "test2" {
  name    = "Test Project 2"
  version = "1.0.0"
}

resource "dependencytrack_policy" "test" {
  name            = "Test Multi Project Policy"
  operator        = "ANY"
  violation_state = "INFO"
}

resource "dependencytrack_project_policy" "test1" {
  policy  = dependencytrack_policy.test.id
  project = dependencytrack_project.test1.id
}

resource "dependencytrack_project_policy" "test2" {
  policy  = dependencytrack_policy.test.id
  project = dependencytrack_project.test2.id
}
`
}
