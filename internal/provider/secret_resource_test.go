package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccSecretResource(t *testing.T) {
	testAccSkipUnlessV5(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSecretResourceConfig("tf-acc-secret", "initial-value", "Initial description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_secret.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("tf-acc-secret"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_secret.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("initial-value"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_secret.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Initial description"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_secret.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("tf-acc-secret"),
					),
				},
			},
			// ImportState testing. The secret value is write-only and cannot
			// be read from the API, so it is excluded from import verification.
			{
				ResourceName:            "dependencytrack_secret.test",
				ImportState:             true,
				ImportStateId:           "tf-acc-secret",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
			// Update and Read testing
			{
				Config: testAccSecretResourceConfig("tf-acc-secret", "updated-value", "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_secret.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("updated-value"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_secret.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Idempotent update: re-applying the same value must succeed even
			// though the server answers 304 Not Modified.
			{
				Config: testAccSecretResourceConfig("tf-acc-secret", "updated-value", "Updated description"),
			},
		},
	})
}

func TestAccSecretResource_NoDescription(t *testing.T) {
	testAccSkipUnlessV5(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecretResourceConfigNoDescription("tf-acc-secret-nodesc", "some-value"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_secret.test",
						tfjsonpath.New("description"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

func testAccSecretResourceConfig(name, value, description string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_secret" "test" {
  name        = %[1]q
  value       = %[2]q
  description = %[3]q
}
`, name, value, description)
}

func testAccSecretResourceConfigNoDescription(name, value string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_secret" "test" {
  name  = %[1]q
  value = %[2]q
}
`, name, value)
}
