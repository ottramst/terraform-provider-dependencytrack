package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLicenseGroupDataSource_ByID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLicenseGroupDataSourceConfigByID,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_license_group.by_id",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-license-group-ds-id"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_license_group.by_id",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_license_group.by_id",
						tfjsonpath.New("risk_weight"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccLicenseGroupDataSource_ByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLicenseGroupDataSourceConfigByName,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_license_group.by_name",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-license-group-ds-name"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_license_group.by_name",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

var testAccLicenseGroupDataSourceConfigByID = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_license_group" "test" {
  name = "tf-acc-license-group-ds-id"
}

data "dependencytrack_license_group" "by_id" {
  id = dependencytrack_license_group.test.id
}
`

var testAccLicenseGroupDataSourceConfigByName = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_license_group" "test" {
  name = "tf-acc-license-group-ds-name"
}

data "dependencytrack_license_group" "by_name" {
  name = dependencytrack_license_group.test.name
}
`
