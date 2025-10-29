package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTeamDataSource_ByUUID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a team resource first, then read it via data source
			{
				Config: testAccTeamDataSourceConfigByUUID,
				ConfigStateChecks: []statecheck.StateCheck{
					// Check the resource was created
					statecheck.ExpectKnownValue(
						"dependencytrack_team.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Team for UUID Lookup"),
					),
					// Check the data source can read it by UUID
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.by_uuid",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Team for UUID Lookup"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.by_uuid",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccTeamDataSource_ByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a team resource first, then read it via data source by name
			{
				Config: testAccTeamDataSourceConfigByName,
				ConfigStateChecks: []statecheck.StateCheck{
					// Check the resource was created
					statecheck.ExpectKnownValue(
						"dependencytrack_team.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Team for Name Lookup"),
					),
					// Check the data source can read it by name
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.by_name",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Team for Name Lookup"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.by_name",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccTeamDataSource_BothUUIDAndName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a team and verify both lookup methods return the same data
			{
				Config: testAccTeamDataSourceConfigBoth,
				ConfigStateChecks: []statecheck.StateCheck{
					// Check the resource was created
					statecheck.ExpectKnownValue(
						"dependencytrack_team.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Team for Both Lookups"),
					),
					// Check data source by UUID
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.by_uuid",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Team for Both Lookups"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.by_uuid",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					// Check data source by name
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.by_name",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Team for Both Lookups"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team.by_name",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

var testAccTeamDataSourceConfigByUUID = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test Team for UUID Lookup"
}

data "dependencytrack_team" "by_uuid" {
  id = dependencytrack_team.test.id
}
`

var testAccTeamDataSourceConfigByName = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test Team for Name Lookup"
}

data "dependencytrack_team" "by_name" {
  name = dependencytrack_team.test.name
}
`

var testAccTeamDataSourceConfigBoth = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test Team for Both Lookups"
}

data "dependencytrack_team" "by_uuid" {
  id = dependencytrack_team.test.id
}

data "dependencytrack_team" "by_name" {
  name = dependencytrack_team.test.name
}
`
