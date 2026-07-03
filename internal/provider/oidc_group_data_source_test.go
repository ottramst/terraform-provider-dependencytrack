package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOIDCGroupDataSource_ByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOIDCGroupDataSourceConfigByName,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_oidc_group.by_name",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-oidc-group-ds"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_oidc_group.by_name",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

var testAccOIDCGroupDataSourceConfigByName = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_oidc_group" "test" {
  name = "tf-acc-oidc-group-ds"
}

data "dependencytrack_oidc_group" "by_name" {
  name = dependencytrack_oidc_group.test.name
}
`
