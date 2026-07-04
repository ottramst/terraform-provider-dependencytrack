package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccProjectMetricsDataSource reads the metrics of a freshly created,
// empty project. On Dependency-Track v4 a fresh project has no metrics
// snapshot at all (the API answers with an empty body), which exercises the
// data source's refresh-and-wait path; v5 synthesizes zeros immediately.
// Either way every counter must come back as zero.
func TestAccProjectMetricsDataSource(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectMetricsDataSourceConfig(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_metrics.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_metrics.test",
						tfjsonpath.New("components"),
						knownvalue.Int64Exact(0),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_metrics.test",
						tfjsonpath.New("vulnerabilities"),
						knownvalue.Int64Exact(0),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_metrics.test",
						tfjsonpath.New("critical"),
						knownvalue.Int64Exact(0),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_metrics.test",
						tfjsonpath.New("findings_total"),
						knownvalue.Int64Exact(0),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_metrics.test",
						tfjsonpath.New("policy_violations_total"),
						knownvalue.Int64Exact(0),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_metrics.test",
						tfjsonpath.New("inherited_risk_score"),
						knownvalue.Float64Exact(0),
					),
				},
			},
		},
	})
}

func testAccProjectMetricsDataSourceConfig(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_project" "test" {
  name    = "tf-acc-project-metrics-%s"
  version = "1.0.0"
}

data "dependencytrack_project_metrics" "test" {
  project = dependencytrack_project.test.id
}
`, suffix)
}
