package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccProjectFindingsDataSource seeds a project with a component that has
// an internal vulnerability assigned (see testAccSeedProjectWithFinding) and
// verifies the data source surfaces the finding, including the CWE IDs whose
// serialization this provider relies on being identical between DT v4 and v5.
func TestAccProjectFindingsDataSource(t *testing.T) {
	projectUUID := testAccSeedProjectWithFinding(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectFindingsDataSourceConfig(projectUUID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(projectUUID),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings").AtSliceIndex(0).AtMapKey("component_name"),
						knownvalue.StringExact("tf-acc-vulnerable-component"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings").AtSliceIndex(0).AtMapKey("component_version"),
						knownvalue.StringExact("1.2.3"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings").AtSliceIndex(0).AtMapKey("source"),
						knownvalue.StringExact("INTERNAL"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings").AtSliceIndex(0).AtMapKey("severity"),
						knownvalue.StringExact("HIGH"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings").AtSliceIndex(0).AtMapKey("cwes"),
						knownvalue.ListExact([]knownvalue.Check{
							knownvalue.Int64Exact(79),
							knownvalue.Int64Exact(89),
						}),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings").AtSliceIndex(0).AtMapKey("is_suppressed"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings").AtSliceIndex(0).AtMapKey("vulnerability_uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings").AtSliceIndex(0).AtMapKey("attributed_on"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

// TestAccProjectFindingsDataSource_Empty verifies the data source returns an
// empty list for a project without findings.
func TestAccProjectFindingsDataSource_Empty(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectFindingsDataSourceEmptyConfig(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_findings.test",
						tfjsonpath.New("findings"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
		},
	})
}

func testAccProjectFindingsDataSourceConfig(projectUUID string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
data "dependencytrack_project_findings" "test" {
  project = %q
}
`, projectUUID)
}

func testAccProjectFindingsDataSourceEmptyConfig(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_project" "test" {
  name    = "tf-acc-no-findings-%s"
  version = "1.0.0"
}

data "dependencytrack_project_findings" "test" {
  project = dependencytrack_project.test.id
}
`, suffix)
}
