package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTagResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTagResourceConfig("tf-acc-tag"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_tag.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-tag"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_tag.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("tf-acc-tag"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Rename (forces replacement, since name has RequiresReplace)
			{
				Config: testAccTagResourceConfig("tf-acc-tag-renamed"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_tag.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-tag-renamed"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTagResourceConfig(name string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_tag" "test" {
  name = %q
}
`, name)
}
