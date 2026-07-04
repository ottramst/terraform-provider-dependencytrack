package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLicenseDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLicenseDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_license.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("TF Acc License DS"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_license.test",
						tfjsonpath.New("osi_approved"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_license.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

var testAccLicenseDataSourceConfig = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_license" "test" {
  license_id   = "tf-acc-license-ds"
  name         = "TF Acc License DS"
  osi_approved = true
}

data "dependencytrack_license" "test" {
  license_id = dependencytrack_license.test.license_id
}
`
