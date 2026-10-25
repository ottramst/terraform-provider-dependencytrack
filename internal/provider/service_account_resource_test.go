package provider

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccServiceAccountResource(t *testing.T) {
	testAccSkipUnlessAtLeast(t, 5, 2)

	name := "acc-sa-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountResourceConfig(name, `email = "ci@example.com"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_service_account.test", tfjsonpath.New("id"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("dependencytrack_service_account.test", tfjsonpath.New("username"), knownvalue.StringExact("svc:"+name)),
					statecheck.ExpectKnownValue("dependencytrack_service_account.test", tfjsonpath.New("email"), knownvalue.StringExact("ci@example.com")),
					statecheck.ExpectKnownValue("dependencytrack_service_account.test", tfjsonpath.New("suspended"), knownvalue.Bool(false)),
				},
			},
			{
				ResourceName:      "dependencytrack_service_account.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Removing the email clears it; suspending is an in-place update.
			{
				Config: testAccServiceAccountResourceConfig(name, `suspended = true`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_service_account.test", tfjsonpath.New("email"), knownvalue.Null()),
					statecheck.ExpectKnownValue("dependencytrack_service_account.test", tfjsonpath.New("suspended"), knownvalue.Bool(true)),
				},
			},
		},
	})
}

func TestAccServiceAccountResource_InvalidName(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccServiceAccountResourceConfig("svc:ci", ""),
				ExpectError: regexp.MustCompile(`reserved "svc:" prefix`),
				PlanOnly:    true,
			},
		},
	})
}

func TestAccServiceAccountResource_Unsupported(t *testing.T) {
	if testAccServerVersion(t).AtLeast(5, 2) {
		t.Skip("test requires a Dependency-Track server older than 5.2")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccServiceAccountResourceConfig("acc-sa-unsupported", ""),
				ExpectError: regexp.MustCompile(`Dependency-Track 5\.2 Required`),
			},
		},
	})
}

func testAccServiceAccountResourceConfig(name, extra string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_service_account" "test" {
  name = %[1]q
  %[2]s
}
`, name, extra)
}

// TestAccServiceAccountAPIKeyResource creates a key, grants the account a
// permission and a team through the existing user resources (keyed by its
// svc: username), and authenticates with the generated key.
func TestAccServiceAccountAPIKeyResource(t *testing.T) {
	testAccSkipUnlessAtLeast(t, 5, 2)

	name := "acc-sa-key-" + randomSuffix()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountAPIKeyResourceConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_service_account_api_key.test", tfjsonpath.New("key"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("dependencytrack_service_account_api_key.test", tfjsonpath.New("public_id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("dependencytrack_service_account_api_key.test", tfjsonpath.New("comment"), knownvalue.StringExact("acceptance test")),
					statecheck.ExpectKnownValue("dependencytrack_service_account_api_key.test", tfjsonpath.New("created_at"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("dependencytrack_service_account_api_key.test", tfjsonpath.New("expires_at"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("dependencytrack_managed_user_permissions.test", tfjsonpath.New("permissions"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact("VIEW_PORTFOLIO")})),
					statecheck.ExpectKnownValue("data.dependencytrack_service_account.test", tfjsonpath.New("permissions"),
						knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact("VIEW_PORTFOLIO")})),
					statecheck.ExpectKnownValue("data.dependencytrack_service_account.test", tfjsonpath.New("teams"),
						knownvalue.ListSizeExact(1)),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountKeyAuthenticates("dependencytrack_service_account_api_key.test"),
				),
			},
			{
				ResourceName:            "dependencytrack_service_account_api_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key", "expires_in_days"},
			},
			{
				ResourceName:      "dependencytrack_managed_user_permissions.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "dependencytrack_user_team_membership.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccServiceAccountAPIKeyResourceConfig(name string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_service_account" "test" {
  name = %[1]q
}

resource "dependencytrack_service_account_api_key" "test" {
  service_account = dependencytrack_service_account.test.name
  comment         = "acceptance test"
  expires_in_days = 7
}

resource "dependencytrack_managed_user_permissions" "test" {
  user        = dependencytrack_service_account.test.username
  permissions = ["VIEW_PORTFOLIO"]
}

resource "dependencytrack_team" "test" {
  name = "%[1]s-team"
}

resource "dependencytrack_user_team_membership" "test" {
  username = dependencytrack_service_account.test.username
  team     = dependencytrack_team.test.id
}

data "dependencytrack_service_account" "test" {
  name = dependencytrack_service_account.test.name

  depends_on = [
    dependencytrack_managed_user_permissions.test,
    dependencytrack_user_team_membership.test,
  ]
}
`, name)
}

// testAccCheckServiceAccountKeyAuthenticates checks that the key in the named
// resource's state authenticates against the API: it may list projects
// (VIEW_PORTFOLIO is granted) but not teams (ACCESS_MANAGEMENT is not).
func testAccCheckServiceAccountKeyAuthenticates(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		key := rs.Primary.Attributes["key"]

		endpoint := strings.TrimSuffix(os.Getenv("DEPENDENCYTRACK_ENDPOINT"), "/")
		for path, want := range map[string]int{"/api/v1/project": http.StatusOK, "/api/v1/team": http.StatusForbidden} {
			req, err := http.NewRequest(http.MethodGet, endpoint+path, nil)
			if err != nil {
				return err
			}
			req.Header.Set("X-Api-Key", key)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			if resp.StatusCode != want {
				return fmt.Errorf("GET %s with the service account key: status %d, want %d", path, resp.StatusCode, want)
			}
		}
		return nil
	}
}
