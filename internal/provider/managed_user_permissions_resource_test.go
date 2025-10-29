package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccManagedUserPermissionsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckUsernamePassword(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccManagedUserPermissionsResourceConfig([]string{"BOM_UPLOAD", "VIEW_PORTFOLIO"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user_permissions.test",
						tfjsonpath.New("user"),
						knownvalue.StringExact("permissions_test_user"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user_permissions.test",
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
				ResourceName:      "dependencytrack_managed_user_permissions.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing - add permission
			{
				Config: testAccManagedUserPermissionsResourceConfig([]string{"BOM_UPLOAD", "VIEW_PORTFOLIO", "PORTFOLIO_MANAGEMENT"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user_permissions.test",
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
				Config: testAccManagedUserPermissionsResourceConfig([]string{"BOM_UPLOAD"}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user_permissions.test",
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

func testAccManagedUserPermissionsResourceConfig(permissions []string) string {
	config := testAccProviderConfigWithUsernamePassword() + `
resource "dependencytrack_managed_user" "test" {
  username = "permissions_test_user"
  fullname = "Permissions Test User"
  email    = "permissions.test@example.com"
  password = "TestPassword123!"
}

resource "dependencytrack_managed_user_permissions" "test" {
  user        = dependencytrack_managed_user.test.id
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

func TestAccManagedUserPermissionsResource_MultiplePermissions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckUsernamePassword(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with many permissions
			{
				Config: testAccManagedUserPermissionsResourceConfigMultiple(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_managed_user_permissions.test",
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

func testAccManagedUserPermissionsResourceConfigMultiple() string {
	return testAccProviderConfigWithUsernamePassword() + `
resource "dependencytrack_managed_user" "test" {
  username = "multi_permissions_test_user"
  fullname = "Multi Permissions Test User"
  email    = "multi.permissions.test@example.com"
  password = "TestPassword123!"
}

resource "dependencytrack_managed_user_permissions" "test" {
  user = dependencytrack_managed_user.test.id
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
