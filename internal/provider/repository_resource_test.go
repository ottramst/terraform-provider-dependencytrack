package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRepositoryResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRepositoryResourceConfig("MAVEN", "tf-acc-maven-repo", "https://example.com/maven", true, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("MAVEN"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("identifier"),
						knownvalue.StringExact("tf-acc-maven-repo"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://example.com/maven"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("enabled"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("internal"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("authentication_required"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("resolution_order"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:            "dependencytrack_repository.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update and Read testing (change url and toggle enabled/internal)
			{
				Config: testAccRepositoryResourceConfig("MAVEN", "tf-acc-maven-repo", "https://example.com/maven-updated", false, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://example.com/maven-updated"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("enabled"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("internal"),
						knownvalue.Bool(true),
					),
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRepositoryResourceConfig(repoType, identifier, url string, enabled, internal bool) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_repository" "test" {
  type       = %[1]q
  identifier = %[2]q
  url        = %[3]q
  enabled    = %[4]t
  internal   = %[5]t
}
`, repoType, identifier, url, enabled, internal)
}

// TestAccRepositoryResource_Authentication exercises the authentication fields
// and the password-preservation behavior on read/import.
//
// It is limited to Dependency-Track v4: on v5 the repository password field is
// interpreted as the name of a stored secret (a literal value is rejected with
// HTTP 400 "The secret with name ... could not be found"), so a literal-password
// round-trip is only valid on v4.
func TestAccRepositoryResource_Authentication(t *testing.T) {
	testAccSkipUnlessV4(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRepositoryResourceConfigAuth("tf-acc-npm-auth", "repouser", "repopass"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("authentication_required"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("repouser"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_repository.test",
						tfjsonpath.New("password"),
						knownvalue.StringExact("repopass"),
					),
				},
			},
			{
				ResourceName:            "dependencytrack_repository.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccRepositoryResourceConfigAuth(identifier, username, password string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_repository" "test" {
  type                    = "NPM"
  identifier              = %[1]q
  url                     = "https://example.com/npm"
  authentication_required = true
  username                = %[2]q
  password                = %[3]q
}
`, identifier, username, password)
}
