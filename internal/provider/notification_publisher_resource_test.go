package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccNotificationPublisherResource(t *testing.T) {
	publisherClass := testAccPublisherClass(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationPublisherResourceConfig(publisherClass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Webhook Publisher"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test",
						tfjsonpath.New("publisher_class"),
						knownvalue.StringExact(publisherClass),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "dependencytrack_notification_publisher.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccNotificationPublisherResourceConfigUpdated(publisherClass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Webhook Publisher Updated"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccNotificationPublisherResourceConfig(publisherClass string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Webhook Publisher"
  description        = "A test webhook notification publisher"
  publisher_class    = %q
  template_mime_type = "application/json"
  template           = "{\"content\": \"test\"}"
}
`, publisherClass)
}

func testAccNotificationPublisherResourceConfigUpdated(publisherClass string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Webhook Publisher Updated"
  description        = "Updated description"
  publisher_class    = %q
  template_mime_type = "application/json"
  template           = "{\"content\": \"updated test\"}"
}
`, publisherClass)
}

func TestAccNotificationPublisherResource_Minimal(t *testing.T) {
	publisherClass := testAccPublisherClass(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with minimal configuration
			{
				Config: testAccNotificationPublisherResourceConfigMinimal(publisherClass),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test_minimal",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Minimal Publisher"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test_minimal",
						tfjsonpath.New("default_publisher"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func testAccNotificationPublisherResourceConfigMinimal(publisherClass string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "test_minimal" {
  name               = "Test Minimal Publisher"
  publisher_class    = %q
  template_mime_type = "text/plain"
}
`, publisherClass)
}
