package provider

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
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
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_policy" "test" {
  name             = %[1]q
  operator         = %[2]q
  violation_state  = %[3]q
}
`, name, operator, violationState)
}

func TestAccPolicyResource_WithConditions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
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
	return testAccProviderConfigWithAPIKey() + `
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
	return testAccProviderConfigWithAPIKey() + `
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

func TestAccPolicyResource_InvertTagMatch(t *testing.T) {
	testAccSkipUnlessAtLeast(t, 5, 2)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfigInvertTagMatch("Inverted Policy", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_policy.test", tfjsonpath.New("invert_tag_match"), knownvalue.Bool(true)),
				},
			},
			{
				ResourceName:      "dependencytrack_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Updating another attribute must not reset the flag.
			{
				Config: testAccPolicyResourceConfigInvertTagMatch("Inverted Policy Renamed", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_policy.test", tfjsonpath.New("invert_tag_match"), knownvalue.Bool(true)),
				},
			},
			{
				Config: testAccPolicyResourceConfigInvertTagMatch("Inverted Policy Renamed", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_policy.test", tfjsonpath.New("invert_tag_match"), knownvalue.Bool(false)),
				},
			},
		},
	})
}

func TestAccPolicyResource_InvertTagMatchUnsupported(t *testing.T) {
	if testAccServerVersion(t).AtLeast(5, 2) {
		t.Skip("test requires a Dependency-Track server older than 5.2")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPolicyResourceConfigInvertTagMatch("Inverted Policy", true),
				ExpectError: regexp.MustCompile(`requires Dependency-Track 5\.2 or newer`),
			},
		},
	})
}

func testAccPolicyResourceConfigInvertTagMatch(name string, invert bool) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_policy" "test" {
  name             = %[1]q
  invert_tag_match = %[2]t
}
`, name, invert)
}

// TestAccPolicyResource_PreservesUnmanagedFields checks that a policy update
// keeps includeChildren and onlyLatestProjectVersion, which this resource
// does not manage but Dependency-Track's update endpoint overwrites from the
// request body.
func TestAccPolicyResource_PreservesUnmanagedFields(t *testing.T) {
	var policyUUID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPolicyResourceConfig("Unmanaged Fields Policy", "ALL", "INFO"),
				Check: func(s *terraform.State) error {
					policyUUID = s.RootModule().Resources["dependencytrack_policy.test"].Primary.ID
					return nil
				},
			},
			{
				// Set the unmanaged fields out of band, as a UI user would.
				PreConfig: func() {
					var policy map[string]any
					if code := testAccAPIDo(t, http.MethodGet, "/api/v1/policy/"+policyUUID, nil, &policy); code != http.StatusOK {
						t.Fatalf("get policy: status %d", code)
					}
					policy["includeChildren"] = true
					policy["onlyLatestProjectVersion"] = true
					body := map[string]any{}
					for _, k := range []string{"uuid", "name", "operator", "violationState", "includeChildren", "onlyLatestProjectVersion"} {
						body[k] = policy[k]
					}
					if code := testAccAPIDo(t, http.MethodPost, "/api/v1/policy", body, nil); code != http.StatusOK {
						t.Fatalf("update policy: status %d", code)
					}
				},
				Config: testAccPolicyResourceConfig("Unmanaged Fields Policy", "ANY", "WARN"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_policy.test", tfjsonpath.New("operator"), knownvalue.StringExact("ANY")),
					statecheck.ExpectKnownValue("dependencytrack_policy.test", tfjsonpath.New("include_children"), knownvalue.Bool(true)),
				},
				Check: func(s *terraform.State) error {
					var policy struct {
						IncludeChildren          bool `json:"includeChildren"`
						OnlyLatestProjectVersion bool `json:"onlyLatestProjectVersion"`
					}
					if code := testAccAPIDo(t, http.MethodGet, "/api/v1/policy/"+policyUUID, nil, &policy); code != http.StatusOK {
						return fmt.Errorf("get policy: status %d", code)
					}
					if !policy.IncludeChildren || !policy.OnlyLatestProjectVersion {
						return fmt.Errorf("update reset unmanaged fields: includeChildren=%t onlyLatestProjectVersion=%t",
							policy.IncludeChildren, policy.OnlyLatestProjectVersion)
					}
					return nil
				},
			},
		},
	})
}
