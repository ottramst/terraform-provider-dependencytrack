package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccExtensionConfigResource(t *testing.T) {
	testAccSkipUnlessV5(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create (adopt) and Read testing: manage the OSS Index analyzer
			// config, which exists with defaults on every v5 instance.
			{
				Config: testAccExtensionConfigResourceConfig("vuln-analyzer", "oss-index", `{
    enabled          = false
    apiUrl           = "https://ossindex.sonatype.org"
    aliasSyncEnabled = false
  }`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_extension_config.test",
						tfjsonpath.New("extension_point"),
						knownvalue.StringExact("vuln-analyzer"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_extension_config.test",
						tfjsonpath.New("extension"),
						knownvalue.StringExact("oss-index"),
					),
					statecheck.ExpectKnownValue(
						"dependencytrack_extension_config.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("vuln-analyzer/oss-index"),
					),
				},
			},
			// ImportState testing. The imported config is the server's
			// serialization, which is semantically but not textually equal to
			// the configured jsonencode() output.
			{
				ResourceName:            "dependencytrack_extension_config.test",
				ImportState:             true,
				ImportStateId:           "vuln-analyzer/oss-index",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config"},
			},
			// Update and Read testing
			{
				Config: testAccExtensionConfigResourceConfig("vuln-analyzer", "oss-index", `{
    enabled          = false
    apiUrl           = "https://ossindex.sonatype.org"
    aliasSyncEnabled = true
  }`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_extension_config.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("vuln-analyzer/oss-index"),
					),
				},
			},
			// Idempotent update: re-applying the same config must succeed
			// even though the server answers 304 Not Modified.
			{
				Config: testAccExtensionConfigResourceConfig("vuln-analyzer", "oss-index", `{
    enabled          = false
    apiUrl           = "https://ossindex.sonatype.org"
    aliasSyncEnabled = true
  }`),
			},
		},
	})
}

// TestAccExtensionConfigResource_SecretRef exercises the x-secret-ref pattern:
// a managed secret referenced by name from an extension config field. The
// GitHub vuln data source stays disabled, so the dummy token is never used.
func TestAccExtensionConfigResource_SecretRef(t *testing.T) {
	testAccSkipUnlessV5(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_secret" "github_token" {
  name  = "tf-acc-github-token"
  value = "dummy-token-value"
}

resource "dependencytrack_extension_config" "github" {
  extension_point = "vuln-data-source"
  extension       = "github"

  config = jsonencode({
    enabled          = false
    aliasSyncEnabled = true
    apiUrl           = "https://api.github.com/graphql"
    apiToken         = dependencytrack_secret.github_token.name
  })
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"dependencytrack_extension_config.github",
						tfjsonpath.New("id"),
						knownvalue.StringExact("vuln-data-source/github"),
					),
				},
			},
		},
	})
}

func TestAccExtensionConfigResource_InvalidConfig(t *testing.T) {
	testAccSkipUnlessV5(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// The server validates the config against the extension's JSON
			// schema; a config missing the required "enabled" field or with
			// unknown fields is rejected with a 400.
			{
				Config: testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_extension_config" "test" {
  extension_point = "vuln-analyzer"
  extension       = "oss-index"

  config = jsonencode({
    definitelyNotAField = true
  })
}
`,
				ExpectError: regexp.MustCompile(`Extension Config Rejected`),
			},
			// Unknown extensions are a 404 mapped to an actionable error.
			{
				Config: testAccProviderConfigWithAPIKey() + `
resource "dependencytrack_extension_config" "test" {
  extension_point = "vuln-analyzer"
  extension       = "does-not-exist"

  config = jsonencode({
    enabled = false
  })
}
`,
				ExpectError: regexp.MustCompile(`Extension Not Found`),
			},
		},
	})
}

func testAccExtensionConfigResourceConfig(extensionPoint, extension, configHCL string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_extension_config" "test" {
  extension_point = %[1]q
  extension       = %[2]q

  config = jsonencode(%[3]s)
}
`, extensionPoint, extension, configHCL)
}
