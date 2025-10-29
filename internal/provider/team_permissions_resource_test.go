package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTeamPermissionsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTeamPermissionsResourceConfig([]string{"BOM_UPLOAD", "VIEW_PORTFOLIO"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_team_permissions.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_team_permissions.test",
						tfjsonpath.New("permissions"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("BOM_UPLOAD"),
							knownvalue.StringExact("VIEW_PORTFOLIO"),
						}),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_team_permissions.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing - add permission
			{
				Config: testAccTeamPermissionsResourceConfig([]string{"BOM_UPLOAD", "VIEW_PORTFOLIO", "PORTFOLIO_MANAGEMENT"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_team_permissions.test",
						tfjsonpath.New("permissions"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("BOM_UPLOAD"),
							knownvalue.StringExact("VIEW_PORTFOLIO"),
							knownvalue.StringExact("PORTFOLIO_MANAGEMENT"),
						}),
					),
				},
			},
			// Update and Read testing - remove permission
			{
				Config: testAccTeamPermissionsResourceConfig([]string{"BOM_UPLOAD"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_team_permissions.test",
						tfjsonpath.New("permissions"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("BOM_UPLOAD"),
						}),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTeamPermissionsResourceConfig(permissions []string) string {
	config := testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Permissions Test Team"
}

resource "dependencytrack_team_permissions" "test" {
  team        = dependencytrack_team.test.id
  permissions = [`

	for i, perm := range permissions {
		if i > 0 {
			config += ", "
		}
		config += `"` + perm + `"`
	}

	config += `]
}
`
	return config
}

func TestAccTeamPermissionsResource_MultiplePermissions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with many permissions
			{
				Config: testAccTeamPermissionsResourceConfigMultiple(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_team_permissions.test",
						tfjsonpath.New("permissions"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("BOM_UPLOAD"),
							knownvalue.StringExact("VIEW_PORTFOLIO"),
							knownvalue.StringExact("VIEW_VULNERABILITY"),
							knownvalue.StringExact("VULNERABILITY_ANALYSIS"),
							knownvalue.StringExact("POLICY_VIOLATION_ANALYSIS"),
						}),
					),
				},
			},
		},
	})
}

func testAccTeamPermissionsResourceConfigMultiple() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Multiple Permissions Test Team"
}

resource "dependencytrack_team_permissions" "test" {
  team = dependencytrack_team.test.id
  permissions = [
    "BOM_UPLOAD",
    "VIEW_PORTFOLIO",
    "VIEW_VULNERABILITY",
    "VULNERABILITY_ANALYSIS",
    "POLICY_VIOLATION_ANALYSIS"
  ]
}
`
}
