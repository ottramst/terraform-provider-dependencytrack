package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccNotificationPublisherDataSource_ByUUID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPublisherDataSourceConfigByUUID,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Publisher for UUID Lookup"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_uuid",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Publisher for UUID Lookup"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_uuid",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_uuid",
						tfjsonpath.New("publisher_class"),
						knownvalue.StringExact("org.dependencytrack.notification.publisher.WebhookPublisher"),
					),
				},
			},
		},
	})
}

func TestAccNotificationPublisherDataSource_ByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPublisherDataSourceConfigByName,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Publisher for Name Lookup"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_name",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Publisher for Name Lookup"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_name",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_name",
						tfjsonpath.New("publisher_class"),
						knownvalue.StringExact("org.dependencytrack.notification.publisher.ConsolePublisher"),
					),
				},
			},
		},
	})
}

func TestAccNotificationPublisherDataSource_BothUUIDAndName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNotificationPublisherDataSourceConfigBoth,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_uuid",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Publisher for Both Lookups"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_uuid",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_name",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Publisher for Both Lookups"),
					),
					statecheck.ExpectKnownValue(
						"data.dependencytrack_notification_publisher.by_name",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

var testAccNotificationPublisherDataSourceConfigByUUID = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Publisher for UUID Lookup"
  description        = "A test publisher for data source UUID lookup"
  publisher_class    = "org.dependencytrack.notification.publisher.WebhookPublisher"
  template_mime_type = "application/json"
  template           = "{\"content\": \"test\"}"
}

data "dependencytrack_notification_publisher" "by_uuid" {
  uuid = dependencytrack_notification_publisher.test.uuid
}
`

var testAccNotificationPublisherDataSourceConfigByName = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Publisher for Name Lookup"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

data "dependencytrack_notification_publisher" "by_name" {
  name = dependencytrack_notification_publisher.test.name
}
`

var testAccNotificationPublisherDataSourceConfigBoth = testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_notification_publisher" "test" {
  name               = "Test Publisher for Both Lookups"
  publisher_class    = "org.dependencytrack.notification.publisher.ConsolePublisher"
  template_mime_type = "text/plain"
}

data "dependencytrack_notification_publisher" "by_uuid" {
  uuid = dependencytrack_notification_publisher.test.uuid
}

data "dependencytrack_notification_publisher" "by_name" {
  name = dependencytrack_notification_publisher.test.name
}
`
