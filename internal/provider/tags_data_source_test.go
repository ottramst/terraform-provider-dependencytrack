package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTagsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTagsDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_tags.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("tags"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_tags.test",
						tfjsonpath.New("tags"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

var testAccTagsDataSourceConfig = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_tag" "test" {
  name = "tf-acc-tags-ds"
}

data "dependencytrack_tags" "test" {
  depends_on = [
    dependencytrack_tag.test
  ]
}
`
