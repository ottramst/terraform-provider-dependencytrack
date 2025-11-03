package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccNotificationRuleProjectResource(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationRuleProjectResourceConfig(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_project.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_project.test",
						tfjsonpath.New("rule"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_project.test",
						tfjsonpath.New("project"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_notification_rule_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccNotificationRuleProjectResourceConfig(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Publisher for Project Association %s"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

resource "dependencytrack_project" "test" {
  name    = "Test Project for Notification Rule %s"
  version = "1.0.0"
}

resource "dependencytrack_notification_rule" "test" {
  name      = "Test Notification Rule for Project %s"
  scope     = "PORTFOLIO"
  publisher = dependencytrack_notification_publisher.test.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]
}

resource "dependencytrack_notification_rule_project" "test" {
  rule    = dependencytrack_notification_rule.test.uuid
  project = dependencytrack_project.test.id
}
`, suffix, suffix, suffix)
}

func TestAccNotificationRuleProjectResource_MultipleProjects(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with multiple projects
			{
				Config: testAccNotificationRuleProjectResourceConfigMultiple(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_project.test1",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_project.test2",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccNotificationRuleProjectResourceConfigMultiple(suffix string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Publisher for Multiple Projects %s"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

resource "dependencytrack_project" "test1" {
  name    = "Test Project 1 for Notification Rule %s"
  version = "1.0.0"
}

resource "dependencytrack_project" "test2" {
  name    = "Test Project 2 for Notification Rule %s"
  version = "1.0.0"
}

resource "dependencytrack_notification_rule" "test" {
  name      = "Test Notification Rule for Multiple Projects %s"
  scope     = "PORTFOLIO"
  publisher = dependencytrack_notification_publisher.test.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]
}

resource "dependencytrack_notification_rule_project" "test1" {
  rule    = dependencytrack_notification_rule.test.uuid
  project = dependencytrack_project.test1.id
}

resource "dependencytrack_notification_rule_project" "test2" {
  rule    = dependencytrack_notification_rule.test.uuid
  project = dependencytrack_project.test2.id
}
`, suffix, suffix, suffix, suffix)
}
