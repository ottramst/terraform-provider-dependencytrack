package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectDataSource_ByUUID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a project resource first, then read it via data source by UUID
			{
				Config: testAccProjectDataSourceConfigByUUID,
				ConfigStateChecks: []statecheck.StateCheck{
					// Check the resource was created
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("DataSource Test Project"),
					),
					// Check the data source can read it by UUID
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("DataSource Test Project"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact("1.0.0"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Test project for data source"),
					),
				},
			},
		},
	})
}

var testAccProjectDataSourceConfigByUUID = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_project" "test" {
  name        = "DataSource Test Project"
  version     = "1.0.0"
  description = "Test project for data source"
}

data "dependencytrack_project" "test" {
  id = dependencytrack_project.test.id
}
`

func TestAccProjectDataSource_ByNameAndVersion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a project resource first, then read it via data source by name and version
			{
				Config: testAccProjectDataSourceConfigByNameAndVersion,
				ConfigStateChecks: []statecheck.StateCheck{
					// Check the resource was created
					statecheck.ExpectKnownValue(
						"dependencytrack_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Lookup Test Project"),
					),
					// Check the data source can read it by name and version
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Lookup Test Project"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact("2.5.0"),
					),
				},
			},
		},
	})
}

var testAccProjectDataSourceConfigByNameAndVersion = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_project" "test" {
  name    = "Lookup Test Project"
  version = "2.5.0"
}

data "dependencytrack_project" "test" {
  name    = dependencytrack_project.test.name
  version = dependencytrack_project.test.version
}
`
