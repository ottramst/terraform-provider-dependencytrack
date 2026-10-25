package provider

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// testAccIssuer is a test token issuer: an RSA key whose public half is
// registered with Dependency-Track as an inline JWKS, so tests can sign
// workload identity tokens without a real OIDC issuer.
type testAccIssuer struct {
	key   *rsa.PrivateKey
	keyID string
}

func newTestAccIssuer(t *testing.T, keyID string) *testAccIssuer {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %s", err)
	}
	return &testAccIssuer{key: key, keyID: keyID}
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// jwks returns the issuer's public key as a JWKS JSON document.
func (i *testAccIssuer) jwks(t *testing.T) string {
	t.Helper()

	b, err := json.Marshal(map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": i.keyID,
		"n": b64url(i.key.N.Bytes()),
		"e": b64url(big.NewInt(int64(i.key.E)).Bytes()),
	}}})
	if err != nil {
		t.Fatalf("marshal JWKS: %s", err)
	}
	return string(b)
}

// token returns an RS256 JWT with the given issuer, subject and audience,
// valid for ten minutes.
func (i *testAccIssuer) token(t *testing.T, iss, sub, aud string) string {
	t.Helper()

	now := time.Now().Unix()
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "typ": "JWT", "kid": i.keyID})
	claims, _ := json.Marshal(map[string]any{"iss": iss, "sub": sub, "aud": aud, "iat": now, "nbf": now, "exp": now + 600})
	signingInput := b64url(header) + "." + b64url(claims)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, i.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign JWT: %s", err)
	}
	return signingInput + "." + b64url(sig)
}

func TestAccWorkloadIdentityProviderResource(t *testing.T) {
	testAccSkipUnlessAtLeast(t, 5, 2)

	name := "acc-wip-" + randomSuffix()
	jwks := newTestAccIssuer(t, "key-1").jwks(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkloadIdentityProviderResourceConfig(name, "aud-1", jwks, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_workload_identity_provider.test", tfjsonpath.New("id"), knownvalue.StringExact(name)),
					statecheck.ExpectKnownValue("dependencytrack_workload_identity_provider.test", tfjsonpath.New("session_lifetime_seconds"), knownvalue.Int64Exact(3600)),
					statecheck.ExpectKnownValue("dependencytrack_workload_identity_provider.test", tfjsonpath.New("jwks_key_ids"),
						knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("key-1")})),
				},
			},
			{
				ResourceName:            "dependencytrack_workload_identity_provider.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"jwks"},
			},
			// Update in place, including rotating the inline keys.
			{
				Config: testAccWorkloadIdentityProviderResourceConfig(name, "aud-2", newTestAccIssuer(t, "key-2").jwks(t), "session_lifetime_seconds = 600"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_workload_identity_provider.test", tfjsonpath.New("audience"), knownvalue.StringExact("aud-2")),
					statecheck.ExpectKnownValue("dependencytrack_workload_identity_provider.test", tfjsonpath.New("session_lifetime_seconds"), knownvalue.Int64Exact(600)),
					statecheck.ExpectKnownValue("dependencytrack_workload_identity_provider.test", tfjsonpath.New("jwks_key_ids"),
						knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("key-2")})),
				},
			},
		},
	})
}

func testAccWorkloadIdentityProviderResourceConfig(name, audience, jwks, extra string) string {
	return testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_workload_identity_provider" "test" {
  name     = %[1]q
  type     = "OIDC"
  issuer   = "https://issuer.example.com"
  audience = %[2]q
  jwks     = %[3]q
  %[4]s
}
`, name, audience, jwks, extra)
}

func TestAccWorkloadIdentityBindingResource(t *testing.T) {
	testAccSkipUnlessAtLeast(t, 5, 2)

	suffix := randomSuffix()
	config := testAccProviderConfigWithAPIKey() + fmt.Sprintf(`
resource "dependencytrack_workload_identity_provider" "test" {
  name     = "acc-wib-%[1]s"
  type     = "OIDC"
  issuer   = "https://issuer.example.com"
  audience = "dependency-track"
  jwks     = %[2]q
}

resource "dependencytrack_service_account" "test" {
  name = "acc-wib-%[1]s"
}

resource "dependencytrack_workload_identity_binding" "test" {
  service_account = dependencytrack_service_account.test.name
  provider_name   = dependencytrack_workload_identity_provider.test.name
  subject         = "repo:acme/app:*"
  condition       = "claims.ref == \"refs/heads/main\""
}
`, suffix, newTestAccIssuer(t, "key-1").jwks(t))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckAPIKey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("dependencytrack_workload_identity_binding.test", tfjsonpath.New("uuid"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue("dependencytrack_workload_identity_binding.test", tfjsonpath.New("subject"), knownvalue.StringExact("repo:acme/app:*")),
				},
			},
			{
				ResourceName:      "dependencytrack_workload_identity_binding.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccWorkloadIdentityFederation authenticates the provider through
// workload identity federation with a token signed by a test issuer. The
// provider, service account and binding are created through the API first:
// in-process acceptance tests share one provider instance between aliases, so
// a single configuration cannot mix API-key and workload identity providers.
func TestAccWorkloadIdentityFederation(t *testing.T) {
	testAccSkipUnlessAtLeast(t, 5, 2)
	testAccSeedPreCheck(t)

	name := "acc-wif-" + randomSuffix()
	issuer := newTestAccIssuer(t, "key-1")

	var jwks map[string]any
	if err := json.Unmarshal([]byte(issuer.jwks(t)), &jwks); err != nil {
		t.Fatalf("unmarshal JWKS: %s", err)
	}
	setup := []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/v2/workload-identity-providers", map[string]any{
			"name": name, "type": "OIDC", "issuer": "https://issuer.example.com", "audience": "dependency-track", "jwks": jwks,
		}},
		{http.MethodPost, "/api/v2/service-accounts", map[string]any{"name": name}},
		{http.MethodPost, "/api/v1/permission/VIEW_PORTFOLIO/user/svc:" + name, nil},
		{http.MethodPost, "/api/v2/service-accounts/" + name + "/workload-identity-bindings", map[string]any{
			"provider_name": name, "subject": "repo:acme/app:*",
		}},
	}
	t.Cleanup(func() {
		testAccAPIDo(t, http.MethodDelete, "/api/v2/service-accounts/"+name, nil, nil)
		testAccAPIDo(t, http.MethodDelete, "/api/v2/workload-identity-providers/"+name, nil, nil)
	})
	for _, s := range setup {
		if code := testAccAPIDo(t, s.method, s.path, s.body, nil); code < 200 || code >= 300 {
			t.Fatalf("%s %s: status %d", s.method, s.path, code)
		}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// A token whose subject no binding matches is refused. It runs first: the
			// post-test destroy reuses the last step's configuration.
			{
				Config: testAccWorkloadIdentityProviderConfig(name,
					issuer.token(t, "https://issuer.example.com", "repo:other/app:ref:refs/heads/main", "dependency-track")),
				ExpectError: regexp.MustCompile(`Unable to exchange the workload identity token`),
			},
			{
				Config: testAccWorkloadIdentityProviderConfig(name,
					issuer.token(t, "https://issuer.example.com", "repo:acme/app:ref:refs/heads/main", "dependency-track")),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.dependencytrack_portfolio_metrics.test", tfjsonpath.New("id"), knownvalue.StringExact("portfolio_metrics")),
				},
			},
		},
	})
}

func testAccWorkloadIdentityProviderConfig(name, token string) string {
	return fmt.Sprintf(`
provider "dependencytrack" {
  endpoint = %[1]q

  workload_identity = {
    provider_name   = %[2]q
    service_account = %[2]q
    token           = %[3]q
  }
}

data "dependencytrack_portfolio_metrics" "test" {}
`, os.Getenv("DEPENDENCYTRACK_ENDPOINT"), name, token)
}
