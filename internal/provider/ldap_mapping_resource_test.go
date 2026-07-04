package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccLDAPMappingResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccLDAPMappingResourceConfig("CN=Developers,OU=Groups,DC=example,DC=com"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_ldap_mapping.test",
						tfjsonpath.New("team"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_ldap_mapping.test",
						tfjsonpath.New("dn"),
						knownvalue.StringExact("CN=Developers,OU=Groups,DC=example,DC=com"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_ldap_mapping.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing using the composite team_uuid/mapping_uuid ID
			{
				ResourceName:      "dependencytrack_ldap_mapping.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["dependencytrack_ldap_mapping.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["team"], rs.Primary.Attributes["id"]), nil
				},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccLDAPMappingResourceConfig(dn string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_team" "test" {
  name = "Test LDAP Mapping Team"
}

resource "dependencytrack_ldap_mapping" "test" {
  team = dependencytrack_team.test.id
  dn   = %[1]q
}
`, dn)
}
