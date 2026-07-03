package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccOIDCGroupResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccOIDCGroupResourceConfig("tf-acc-oidc-group"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_oidc_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-oidc-group"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_oidc_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_oidc_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccOIDCGroupResourceConfig("tf-acc-oidc-group-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_oidc_group.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-oidc-group-updated"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccOIDCGroupResourceConfig(name string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_oidc_group" "test" {
  name = %q
}
`, name)
}
