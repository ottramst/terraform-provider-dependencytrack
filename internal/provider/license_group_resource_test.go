package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLicenseGroupResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccLicenseGroupResourceConfig("tf-acc-license-group"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_license_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-license-group"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license_group.test",
						tfjsonpath.New("risk_weight"),
						knownvalue.Int64Exact(0),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_license_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccLicenseGroupResourceConfig("tf-acc-license-group-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_license_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-license-group-updated"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccLicenseGroupResourceConfig(name string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_license_group" "test" {
  name = %q
}
`, name)
}
