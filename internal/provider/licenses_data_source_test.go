package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLicensesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLicensesDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_licenses.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("licenses"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_licenses.test",
						tfjsonpath.New("licenses"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

var testAccLicensesDataSourceConfig = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_license" "test" {
  license_id = "tf-acc-licenses-ds"
  name       = "TF Acc Licenses DS"
}

data "dependencytrack_licenses" "test" {
  depends_on = [
    dependencytrack_license.test
  ]
}
`
