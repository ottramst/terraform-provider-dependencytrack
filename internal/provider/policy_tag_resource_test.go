package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPolicyTagResource(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyTagResourceConfig(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_policy_tag.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_policy_tag.test",
						tfjsonpath.New("tag"),
						knownvalue.StringExact(fmt.Sprintf("tf-acc-policy-tag-%s", suffix)),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_policy_tag.test",
						tfjsonpath.New("policy"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_policy_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccPolicyTagResourceConfig(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_tag" "test" {
  name = "tf-acc-policy-tag-%s"
}

resource "dependencytrack_policy" "test" {
  name            = "tf-acc-tagged-policy-%s"
  operator        = "ANY"
  violation_state = "WARN"
}

resource "dependencytrack_policy_tag" "test" {
  tag    = dependencytrack_tag.test.name
  policy = dependencytrack_policy.test.id
}
`, suffix, suffix)
}
