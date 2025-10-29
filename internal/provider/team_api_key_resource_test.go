package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTeamAPIKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamAPIKeyResourceConfig("Test API Key"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test",
						tfjsonpath.New("key"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test",
						tfjsonpath.New("comment"),
						knownvalue.StringExact("Test API Key"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test",
						tfjsonpath.New("masked_key"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update comment and Read testing
			{
				Config: testAccTeamAPIKeyResourceConfig("Updated API Key Comment"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test",
						tfjsonpath.New("comment"),
						knownvalue.StringExact("Updated API Key Comment"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTeamAPIKeyResourceConfig(comment string) string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test Team API Key"
}

resource "dependencytrack_team_api_key" "test" {
  team    = dependencytrack_team.test.id
  comment = "` + comment + `"
}
`
}

func TestAccTeamAPIKeyResource_NoComment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without comment
			{
				Config: testAccTeamAPIKeyResourceConfigNoComment(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test",
						tfjsonpath.New("key"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccTeamAPIKeyResourceConfigNoComment() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test Team API Key No Comment"
}

resource "dependencytrack_team_api_key" "test" {
  team = dependencytrack_team.test.id
}
`
}

func TestAccTeamAPIKeyResource_Multiple(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple API keys for the same team
			{
				Config: testAccTeamAPIKeyResourceConfigMultiple(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test1",
						tfjsonpath.New("comment"),
						knownvalue.StringExact("First API Key"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_team_api_key.test2",
						tfjsonpath.New("comment"),
						knownvalue.StringExact("Second API Key"),
					),
				},
			},
		},
	})
}

func testAccTeamAPIKeyResourceConfigMultiple() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test Team Multiple Keys"
}

resource "dependencytrack_team_api_key" "test1" {
  team    = dependencytrack_team.test.id
  comment = "First API Key"
}

resource "dependencytrack_team_api_key" "test2" {
  team    = dependencytrack_team.test.id
  comment = "Second API Key"
}
`
}
