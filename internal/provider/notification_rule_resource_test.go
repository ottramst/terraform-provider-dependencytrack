package provider

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func randomSuffix() string {
	return fmt.Sprintf("%d", rand.IntN(100000))
}

func TestAccNotificationRuleResource(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationRuleResourceConfig(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Notification Rule"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test",
						tfjsonpath.New("scope"),
						knownvalue.StringExact("PORTFOLIO"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test",
						tfjsonpath.New("enabled"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test",
						tfjsonpath.New("notify_on"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("NEW_VULNERABILITY"),
							knownvalue.StringExact("NEW_VULNERABLE_DEPENDENCY"),
						}),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_notification_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccNotificationRuleResourceConfigUpdated(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Notification Rule Updated"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test",
						tfjsonpath.New("enabled"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test",
						tfjsonpath.New("notify_on"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("NEW_VULNERABILITY"),
							knownvalue.StringExact("POLICY_VIOLATION"),
						}),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccNotificationRuleResourceConfig(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Publisher for Rule %s"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

resource "dependencytrack_project" "test" {
  name    = "Test Notification Rule Project %s"
  version = "1.0.0"
}

resource "dependencytrack_notification_rule" "test" {
  name              = "Test Notification Rule"
  scope             = "PORTFOLIO"
  notification_level = "INFORMATIONAL"
  publisher         = dependencytrack_notification_publisher.test.id

  notify_on = [
    "NEW_VULNERABILITY",
    "NEW_VULNERABLE_DEPENDENCY"
  ]

  enabled               = true
  notify_children       = false
  log_successful_publish = false
}
`, suffix, suffix)
}

func testAccNotificationRuleResourceConfigUpdated(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Publisher for Rule %s"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

resource "dependencytrack_project" "test" {
  name    = "Test Notification Rule Project %s"
  version = "1.0.0"
}

resource "dependencytrack_notification_rule" "test" {
  name              = "Test Notification Rule Updated"
  scope             = "PORTFOLIO"
  notification_level = "WARNING"
  publisher         = dependencytrack_notification_publisher.test.id

  notify_on = [
    "NEW_VULNERABILITY",
    "POLICY_VIOLATION"
  ]

  enabled               = false
  notify_children       = true
  log_successful_publish = true
}
`, suffix, suffix)
}

func TestAccNotificationRuleResource_Minimal(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with minimal configuration
			{
				Config: testAccNotificationRuleResourceConfigMinimal(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test_minimal",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Minimal Notification Rule"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test_minimal",
						tfjsonpath.New("scope"),
						knownvalue.StringExact("SYSTEM"),
					),
				},
			},
		},
	})
}

func testAccNotificationRuleResourceConfigMinimal(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test_minimal" {
  name               = "Test Publisher Minimal %s"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

resource "dependencytrack_notification_rule" "test_minimal" {
  name      = "Minimal Notification Rule"
  scope     = "SYSTEM"
  publisher = dependencytrack_notification_publisher.test_minimal.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]
}
`, suffix)
}

func TestAccNotificationRuleResource_WithTeams(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with teams
			{
				Config: testAccNotificationRuleResourceConfigWithTeams(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule.test_teams",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Notification Rule with Teams"),
					),
				},
			},
		},
	})
}

func testAccNotificationRuleResourceConfigWithTeams(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test_teams" {
  name               = "Test Publisher with Teams %s"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

resource "dependencytrack_team" "test" {
  name = "Test Notification Team %s"
}

resource "dependencytrack_notification_rule" "test_teams" {
  name      = "Notification Rule with Teams"
  scope     = "SYSTEM"
  publisher = dependencytrack_notification_publisher.test_teams.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]
}
`, suffix, suffix)
}
