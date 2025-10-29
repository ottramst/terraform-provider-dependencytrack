package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccUserTeamMembershipResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccUserTeamMembershipResourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_user_team_membership.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("membership-test-user"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_user_team_membership.test",
						tfjsonpath.New("team_uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_user_team_membership.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_user_team_membership.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccUserTeamMembershipResourceConfig() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "User Membership Test Team"
}

resource "dependencytrack_managed_user" "test" {
  username = "membership-test-user"
  fullname = "Membership Test User"
  email    = "membership-test@example.com"
  password = "Test123!@#"
}

resource "dependencytrack_user_team_membership" "test" {
  username  = dependencytrack_managed_user.test.username
  team_uuid = dependencytrack_team.test.id
}
`
}

func TestAccUserTeamMembershipResource_MultipleMemberships(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create user with multiple team memberships
			{
				Config: testAccUserTeamMembershipResourceConfigMultiple(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_user_team_membership.test1",
						tfjsonpath.New("username"),
						knownvalue.StringExact("multi-membership-test-user"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_user_team_membership.test2",
						tfjsonpath.New("username"),
						knownvalue.StringExact("multi-membership-test-user"),
					),
				},
			},
		},
	})
}

func testAccUserTeamMembershipResourceConfigMultiple() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test1" {
  name = "Multi Membership Test Team 1"
}

resource "dependencytrack_team" "test2" {
  name = "Multi Membership Test Team 2"
}

resource "dependencytrack_managed_user" "test" {
  username = "multi-membership-test-user"
  fullname = "Multi Membership Test User"
  email    = "multi-membership-test@example.com"
  password = "Test123!@#"
}

resource "dependencytrack_user_team_membership" "test1" {
  username  = dependencytrack_managed_user.test.username
  team_uuid = dependencytrack_team.test1.id
}

resource "dependencytrack_user_team_membership" "test2" {
  username  = dependencytrack_managed_user.test.username
  team_uuid = dependencytrack_team.test2.id
}
`
}

func TestAccUserTeamMembershipResource_Replace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create initial membership
			{
				Config: testAccUserTeamMembershipResourceConfigReplace("team1"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_user_team_membership.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("replace-test-user"),
					),
				},
			},
			// Replace with different team (should recreate resource)
			{
				Config: testAccUserTeamMembershipResourceConfigReplace("team2"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_user_team_membership.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("replace-test-user"),
					),
				},
			},
		},
	})
}

func testAccUserTeamMembershipResourceConfigReplace(teamRef string) string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "team1" {
  name = "Replace Test Team 1"
}

resource "dependencytrack_team" "team2" {
  name = "Replace Test Team 2"
}

resource "dependencytrack_managed_user" "test" {
  username = "replace-test-user"
  fullname = "Replace Test User"
  email    = "replace-test@example.com"
  password = "Test123!@#"
}

resource "dependencytrack_user_team_membership" "test" {
  username  = dependencytrack_managed_user.test.username
  team_uuid = dependencytrack_team.` + teamRef + `.id
}
`
}
