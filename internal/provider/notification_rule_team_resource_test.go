package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccNotificationRuleTeamResource(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationRuleTeamResourceConfig(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_team.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_team.test",
						tfjsonpath.New("rule"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_team.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_notification_rule_team.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccNotificationRuleTeamResourceConfig(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Email Publisher for Team Association %s"
  publisher_class    = "org.dependencytrack.notification.publisher.SendMailPublisher"
  template_mime_type = "text/plain"
  description        = "Test email publisher for team notifications"
}

resource "dependencytrack_team" "test" {
  name = "Test Team for Notification Rule %s"
}

resource "dependencytrack_notification_rule" "test" {
  name      = "Test Notification Rule for Team %s"
  scope     = "SYSTEM"
  publisher = dependencytrack_notification_publisher.test.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]
}

resource "dependencytrack_notification_rule_team" "test" {
  rule = dependencytrack_notification_rule.test.uuid
  team = dependencytrack_team.test.id
}
`, suffix, suffix, suffix)
}

func TestAccNotificationRuleTeamResource_MultipleTeams(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with multiple teams
			{
				Config: testAccNotificationRuleTeamResourceConfigMultiple(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_team.test1",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_team.test2",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccNotificationRuleTeamResourceConfigMultiple(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Email Publisher for Multiple Teams %s"
  publisher_class    = "org.dependencytrack.notification.publisher.SendMailPublisher"
  template_mime_type = "text/plain"
  description        = "Test email publisher for multiple team notifications"
}

resource "dependencytrack_team" "test1" {
  name = "Test Team 1 for Notification Rule %s"
}

resource "dependencytrack_team" "test2" {
  name = "Test Team 2 for Notification Rule %s"
}

resource "dependencytrack_notification_rule" "test" {
  name      = "Test Notification Rule for Multiple Teams %s"
  scope     = "SYSTEM"
  publisher = dependencytrack_notification_publisher.test.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]
}

resource "dependencytrack_notification_rule_team" "test1" {
  rule = dependencytrack_notification_rule.test.uuid
  team = dependencytrack_team.test1.id
}

resource "dependencytrack_notification_rule_team" "test2" {
  rule = dependencytrack_notification_rule.test.uuid
  team = dependencytrack_team.test2.id
}
`, suffix, suffix, suffix, suffix)
}
