package provider

import (
	"fmt"
	"net/http"
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

// TestAccNotificationPublisherResource_RecreateAfterExternalDelete exercises
// the I2 fix: getPublisher reports a missing publisher via found=false (there
// is no get-by-uuid endpoint, so a list-based getter can't surface an HTTP
// 404), and Read must translate that into RemoveResource so a subsequent apply
// recreates the resource. Step 2 deletes the publisher out-of-band and re-runs
// the same config; the mid-step refresh must silently drop it from state and
// plan a create. With the old unreachable isNotFound check the refresh would
// hard-error instead.
func TestAccNotificationPublisherResource_RecreateAfterExternalDelete(t *testing.T) {
	publisherClass := testAccPublisherClass(t)
	const name = "Test Publisher Recreate"
	config := testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_notification_publisher" "recreate" {
  name               = %q
  publisher_class    = %q
  template_mime_type = "application/json"
  template           = "{\"content\": \"test\"}"
}
`, name, publisherClass)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.recreate",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				PreConfig: func() { testAccDeleteNotificationPublisherByName(t, name) },
				Config:    config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_notification_publisher.recreate",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
		},
	})
}

// testAccDeleteNotificationPublisherByName deletes the notification publisher
// with the given name directly via the API, simulating an out-of-band deletion.
func testAccDeleteNotificationPublisherByName(t *testing.T, name string) {
	t.Helper()

	var publishers []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}
	if code := testAccAPIDo(t, http.MethodGet, "/api/v1/notification/publisher?pageSize=100", nil, &publishers); code < 200 || code >= 300 {
		t.Fatalf("list notification publishers: status %d", code)
	}

	for _, p := range publishers {
		if p.Name == name {
			if code := testAccAPIDo(t, http.MethodDelete, "/api/v1/notification/publisher/"+p.UUID, nil, nil); code < 200 || code >= 300 {
				t.Fatalf("delete notification publisher %s: status %d", p.UUID, code)
			}
			return
		}
	}

	t.Fatalf("notification publisher %q not found for external delete", name)
}
