package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOIDCGroupMappingResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccOIDCGroupMappingResourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_oidc_group_mapping.test",
						tfjsonpath.New("group"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_oidc_group_mapping.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_oidc_group_mapping.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccOIDCGroupMappingResourceConfig() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_team" "test" {
  name = "Test OIDC Mapping Team"
}

resource "dependencytrack_oidc_group" "test" {
  name = "tf-acc-oidc-mapping-group"
}

resource "dependencytrack_oidc_group_mapping" "test" {
  group = dependencytrack_oidc_group.test.id
  team  = dependencytrack_team.test.id
}
`
}
