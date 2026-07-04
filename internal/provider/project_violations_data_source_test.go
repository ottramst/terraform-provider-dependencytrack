package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccProjectViolationsDataSource seeds a project with a component that
// violates an operational policy (see testAccSeedProjectWithViolation) and
// verifies the data source surfaces the violation.
func TestAccProjectViolationsDataSource(t *testing.T) {
	projectUUID := testAccSeedProjectWithViolation(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectViolationsDataSourceConfig(projectUUID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(projectUUID),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("violations"),
						knownvalue.ListSizeExact(1),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("violations").AtSliceIndex(0).AtMapKey("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("violations").AtSliceIndex(0).AtMapKey("type"),
						knownvalue.StringExact("OPERATIONAL"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("violations").AtSliceIndex(0).AtMapKey("policy_violation_state"),
						knownvalue.StringExact("FAIL"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("violations").AtSliceIndex(0).AtMapKey("component_name"),
						knownvalue.StringExact("tf-acc-violating-component"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("violations").AtSliceIndex(0).AtMapKey("component_version"),
						knownvalue.StringExact("4.5.6"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("violations").AtSliceIndex(0).AtMapKey("component_uuid"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

// TestAccProjectViolationsDataSource_Empty verifies the data source returns
// an empty list for a project without violations.
func TestAccProjectViolationsDataSource_Empty(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectViolationsDataSourceEmptyConfig(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_project_violations.test",
						tfjsonpath.New("violations"),
						knownvalue.ListSizeExact(0),
					),
				},
			},
		},
	})
}

func testAccProjectViolationsDataSourceConfig(projectUUID string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
data "dependencytrack_project_violations" "test" {
  project = %q
}
`, projectUUID)
}

func testAccProjectViolationsDataSourceEmptyConfig(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_project" "test" {
  name    = "tf-acc-no-violations-%s"
  version = "1.0.0"
}

data "dependencytrack_project_violations" "test" {
  project = dependencytrack_project.test.id
}
`, suffix)
}
