package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPolicyDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckUsernamePassword(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyDataSourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_policy.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Data Source Test Policy"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_policy.test",
						tfjsonpath.New("operator"),
						knownvalue.StringExact("ALL"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_policy.test",
						tfjsonpath.New("violation_state"),
						knownvalue.StringExact("INFO"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_policy.test",
						tfjsonpath.New("global"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_policy.test",
						tfjsonpath.New("include_children"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func testAccPolicyDataSourceConfig() string {
	return testAccProviderConfigWithUsernamePassword() + `
resource "dependencytrack_policy" "test" {
  name            = "Data Source Test Policy"
  operator        = "ALL"
  violation_state = "INFO"
}

data "dependencytrack_policy" "test" {
  id = dependencytrack_policy.test.id
}
`
}
