package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLicenseGroupLicenseResource(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccLicenseGroupLicenseResourceConfig(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_license_group_license.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license_group_license.test",
						tfjsonpath.New("license_group"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license_group_license.test",
						tfjsonpath.New("license"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_license_group_license.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccLicenseGroupLicenseResourceConfig(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_license_group" "test" {
  name = "tf-acc-license-group-membership-%s"
}

resource "dependencytrack_license" "test" {
  license_id = "tf-acc-grouped-license-%s"
  name       = "TF Acc Grouped License %s"
}

resource "dependencytrack_license_group_license" "test" {
  license_group = dependencytrack_license_group.test.id
  license       = dependencytrack_license.test.uuid
}
`, suffix, suffix, suffix)
}
