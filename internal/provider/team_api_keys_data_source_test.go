package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTeamAPIKeysDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccTeamAPIKeysDataSourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team_api_keys.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team_api_keys.test",
						tfjsonpath.New("api_keys"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccTeamAPIKeysDataSourceConfig() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test Team API Keys Data"
}

resource "dependencytrack_team_api_key" "test1" {
  team    = dependencytrack_team.test.id
  comment = "Data Source Test Key 1"
}

resource "dependencytrack_team_api_key" "test2" {
  team    = dependencytrack_team.test.id
  comment = "Data Source Test Key 2"
}

data "dependencytrack_team_api_keys" "test" {
  team = dependencytrack_team.test.id
  depends_on = [
    dependencytrack_team_api_key.test1,
    dependencytrack_team_api_key.test2
  ]
}
`
}

func TestAccTeamAPIKeysDataSource_Empty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing with empty API keys
			{
				Config: testAccTeamAPIKeysDataSourceConfigEmpty(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team_api_keys.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_team_api_keys.test",
						tfjsonpath.New("api_keys"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
		},
	})
}

func testAccTeamAPIKeysDataSourceConfigEmpty() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test Team API Keys Empty"
}

data "dependencytrack_team_api_keys" "test" {
  team = dependencytrack_team.test.id
}
`
}
