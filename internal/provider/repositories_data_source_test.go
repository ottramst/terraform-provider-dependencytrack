package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRepositoriesDataSource_ByType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoriesDataSourceConfigByType,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_repositories.by_type",
						tfjsonpath.New("id"),
						knownvalue.StringExact("MAVEN"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_repositories.by_type",
						tfjsonpath.New("repositories"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccRepositoriesDataSource_All(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoriesDataSourceConfigAll,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_repositories.all",
						tfjsonpath.New("id"),
						knownvalue.StringExact("all"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_repositories.all",
						tfjsonpath.New("repositories"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

var testAccRepositoriesDataSourceConfigByType = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_repository" "test" {
  type       = "MAVEN"
  identifier = "tf-acc-repos-ds-maven"
  url        = "https://example.com/maven"
}

data "dependencytrack_repositories" "by_type" {
  type = "MAVEN"
  depends_on = [
    dependencytrack_repository.test
  ]
}
`

var testAccRepositoriesDataSourceConfigAll = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_repository" "test" {
  type       = "NPM"
  identifier = "tf-acc-repos-ds-npm"
  url        = "https://example.com/npm"
}

data "dependencytrack_repositories" "all" {
  depends_on = [
    dependencytrack_repository.test
  ]
}
`
