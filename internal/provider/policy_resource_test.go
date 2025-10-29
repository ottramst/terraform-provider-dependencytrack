package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckUsernamePassword(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPolicyResourceConfig("Test Policy", "ALL", "INFO"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Policy"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("operator"),
						knownvalue.StringExact("ALL"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("violation_state"),
						knownvalue.StringExact("INFO"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccPolicyResourceConfig("Updated Policy", "ANY", "WARN"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Updated Policy"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("operator"),
						knownvalue.StringExact("ANY"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("violation_state"),
						knownvalue.StringExact("WARN"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccPolicyResourceConfig(name, operator, violationState string) string {
	return testAccProviderConfigWithUsernamePassword() + fmt.Sprintf(`
resource "dependencytrack_policy" "test" {
  name             = %[1]q
  operator         = %[2]q
  violation_state  = %[3]q
}
`, name, operator, violationState)
}

func TestAccPolicyResource_WithConditions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckUsernamePassword(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with conditions
			{
				Config: testAccPolicyResourceConfigWithConditions(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Policy with Conditions"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("operator"),
						knownvalue.StringExact("ANY"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("violation_state"),
						knownvalue.StringExact("WARN"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update conditions
			{
				Config: testAccPolicyResourceConfigWithUpdatedConditions(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_policy.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Policy with Conditions"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccPolicyResourceConfigWithConditions() string {
	return testAccProviderConfigWithUsernamePassword() + `
resource "dependencytrack_policy" "test" {
  name             = "Policy with Conditions"
  operator         = "ANY"
  violation_state  = "WARN"

  conditions = [
    {
      subject  = "SEVERITY"
      operator = "IS"
      value    = "CRITICAL"
    },
    {
      subject  = "LICENSE"
      operator = "IS"
      value    = "GPL-3.0"
    }
  ]
}
`
}

func testAccPolicyResourceConfigWithUpdatedConditions() string {
	return testAccProviderConfigWithUsernamePassword() + `
resource "dependencytrack_policy" "test" {
  name             = "Policy with Conditions"
  operator         = "ALL"
  violation_state  = "FAIL"

  conditions = [
    {
      subject  = "SEVERITY"
      operator = "IS"
      value    = "HIGH"
    }
  ]
}
`
}
