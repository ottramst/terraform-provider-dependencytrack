package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPortfolioMetricsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortfolioMetricsDataSourceConfig(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_portfolio_metrics.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("portfolio_metrics"),
					),
					// Absolute values depend on what other tests have created
					// on the shared instance, so only assert presence.
					statecheck.ExpectKnownValue(
						"data.dependencytrack_portfolio_metrics.test",
						tfjsonpath.New("projects"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_portfolio_metrics.test",
						tfjsonpath.New("components"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_portfolio_metrics.test",
						tfjsonpath.New("vulnerabilities"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_portfolio_metrics.test",
						tfjsonpath.New("inherited_risk_score"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_portfolio_metrics.test",
						tfjsonpath.New("policy_violations_total"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccPortfolioMetricsDataSourceConfig() string {
	return testAccProviderConfigWithAPIKey() + `
data "dependencytrack_portfolio_metrics" "test" {}
`
}
