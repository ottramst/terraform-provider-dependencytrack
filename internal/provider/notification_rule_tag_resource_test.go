package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccNotificationRuleTagResource(t *testing.T) {
	suffix := randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationRuleTagResourceConfig(suffix, testAccPublisherClass(t)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_tag.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_tag.test",
						tfjsonpath.New("tag"),
						knownvalue.StringExact(fmt.Sprintf("tf-acc-rule-tag-%s", suffix)),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_rule_tag.test",
						tfjsonpath.New("notification_rule"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_notification_rule_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccNotificationRuleTagResourceConfig(suffix, publisherClass string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_tag" "test" {
  name = "tf-acc-rule-tag-%s"
}

resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Publisher for Rule Tag %s"
  publisher_class    = %q
  template_mime_type = "application/json"
  description        = "Test publisher for notification rule tagging"
}

resource "dependencytrack_notification_rule" "test" {
  name      = "Test Notification Rule for Tag %s"
  scope     = "PORTFOLIO"
  publisher = dependencytrack_notification_publisher.test.id

  notify_on = [
    "NEW_VULNERABILITY"
  ]
}

resource "dependencytrack_notification_rule_tag" "test" {
  tag               = dependencytrack_tag.test.name
  notification_rule = dependencytrack_notification_rule.test.uuid
}
`, suffix, suffix, publisherClass, suffix)
}
