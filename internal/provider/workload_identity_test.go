package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestExchangeWorkloadIdentityToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/oauth/token" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %s", err)
		}
		want := map[string]string{
			"grant_type":                 tokenExchangeGrantType,
			"subject_token":              "the-jwt",
			"subject_token_type":         tokenTypeJWT,
			"workload_identity_provider": "github-actions",
			"service_account":            "ci",
		}
		for k, v := range want {
			if got := r.PostForm.Get(k); got != v {
				t.Errorf("form %s = %q, want %q", k, got, v)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"session-token","issued_token_type":"urn:ietf:params:oauth:token-type:access_token","token_type":"Bearer","expires_in":600}`))
	}))
	defer srv.Close()

	token, expiresIn, err := exchangeWorkloadIdentityToken(context.Background(), srv.URL+"/", "github-actions", "ci", "the-jwt")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if token != "session-token" || expiresIn != 600 {
		t.Errorf("got (%q, %d), want (session-token, 600)", token, expiresIn)
	}
}

func TestExchangeWorkloadIdentityToken_Errors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"oauth error", http.StatusBadRequest, `{"error":"invalid_request","error_description":"The request was refused"}`, "invalid_request: The request was refused"},
		{"problem details", http.StatusServiceUnavailable, `{"title":"Service Unavailable","status":503}`, "status 503"},
		{"missing token", http.StatusOK, `{"token_type":"Bearer"}`, "no access_token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			_, _, err := exchangeWorkloadIdentityToken(context.Background(), srv.URL, "p", "sa", "jwt")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestProviderConfig_WorkloadIdentityConflictsWithAPIKey(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "dependencytrack" {
  endpoint = "http://127.0.0.1:1"
  api_key  = "key"

  workload_identity = {
    provider_name   = "github-actions"
    service_account = "ci"
    token           = "jwt"
  }
}

data "dependencytrack_portfolio_metrics" "test" {}
`,
				ExpectError: regexp.MustCompile(`workload_identity cannot be combined with api_key`),
			},
		},
	})
}
