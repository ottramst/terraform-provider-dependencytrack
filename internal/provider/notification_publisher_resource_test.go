package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccNotificationPublisherResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccNotificationPublisherResourceConfig(),
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
						knownvalue.StringExact("org.dependencytrack.notification.publisher.WebhookPublisher"),
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
				Config: testAccNotificationPublisherResourceConfigUpdated(),
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

func testAccNotificationPublisherResourceConfig() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Webhook Publisher"
  description        = "A test webhook notification publisher"
  publisher_class    = "org.dependencytrack.notification.publisher.WebhookPublisher"
  template_mime_type = "application/json"
  template           = "{\"content\": \"test\"}"
}
`
}

func testAccNotificationPublisherResourceConfigUpdated() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Webhook Publisher Updated"
  description        = "Updated description"
  publisher_class    = "org.dependencytrack.notification.publisher.WebhookPublisher"
  template_mime_type = "application/json"
  template           = "{\"content\": \"updated test\"}"
}
`
}

func TestAccNotificationPublisherResource_Minimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with minimal configuration
			{
				Config: testAccNotificationPublisherResourceConfigMinimal(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test_minimal",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Console Publisher"),
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

func testAccNotificationPublisherResourceConfigMinimal() string {
	return testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_notification_publisher" "test_minimal" {
  name               = "Test Console Publisher"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}
`
}
