package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLicenseResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccLicenseResourceConfig("tf-acc-license", "TF Acc License"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_license.test",
						tfjsonpath.New("license_id"),
						knownvalue.StringExact("tf-acc-license"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc License"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license.test",
						tfjsonpath.New("osi_approved"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license.test",
						tfjsonpath.New("fsf_libre"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license.test",
						tfjsonpath.New("see_also"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.StringExact("https://example.com/a"),
							knownvalue.StringExact("https://example.com/b"),
						}),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_license.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing (import by license_id)
			{
				ResourceName:      "dependencytrack_license.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update forces replacement (no update endpoint): change the name
			{
				Config: testAccLicenseResourceConfig("tf-acc-license", "TF Acc License Renamed"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_license.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc License Renamed"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccLicenseResourceConfig(licenseID, name string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_license" "test" {
  license_id   = %[1]q
  name         = %[2]q
  text         = "The license text."
  comment      = "A custom license for acceptance testing."
  osi_approved = true
  fsf_libre    = true
  see_also = [
    "https://example.com/a",
    "https://example.com/b",
  ]
}
`, licenseID, name)
}
