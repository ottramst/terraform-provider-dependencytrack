package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestParseCompositeID3(t *testing.T) {
	tests := []struct {
		name                   string
		id                     string
		wantP1, wantP2, wantP3 string
		wantErr                bool
	}{
		{"three parts", "a/b/c", "a", "b", "c", false},
		{"uuid group name", "00000000-0000-0000-0000-000000000001/general/color", "00000000-0000-0000-0000-000000000001", "general", "color", false},
		{"empty parts allowed", "//", "", "", "", false},
		{"too few parts", "a/b", "", "", "", true},
		{"too many parts", "a/b/c/d", "", "", "", true},
		{"single part", "a", "", "", "", true},
		{"empty string", "", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p1, p2, p3, err := parseCompositeID3(tt.id, "project", "group", "name")
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseCompositeID3(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if p1 != tt.wantP1 || p2 != tt.wantP2 || p3 != tt.wantP3 {
				t.Errorf("parseCompositeID3(%q) = (%q, %q, %q), want (%q, %q, %q)", tt.id, p1, p2, p3, tt.wantP1, tt.wantP2, tt.wantP3)
			}
		})
	}
}

func TestJSONStringsEquivalent(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical", `{"a":1}`, `{"a":1}`, true},
		{"whitespace differences", `{"destinationUrl":"https://example.com"}`, `{"destinationUrl": "https://example.com"}`, true},
		{"key order differences", `{"a":1,"b":2}`, `{"b":2,"a":1}`, true},
		{"different values", `{"a":1}`, `{"a":2}`, false},
		{"different keys", `{"destination":"x"}`, `{"destinationUrl":"x"}`, false},
		{"invalid json falls back to string equality (equal)", "not-json", "not-json", true},
		{"invalid json falls back to string equality (unequal)", "not-json", `{"a":1}`, false},
		{"empty vs json", "", `{"a":1}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jsonStringsEquivalent(tt.a, tt.b); got != tt.want {
				t.Errorf("jsonStringsEquivalent(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCanonicalJSONString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"already canonical", `{"a":1}`, `{"a":1}`},
		{"strips whitespace", `{"destinationUrl": "https://example.com"}`, `{"destinationUrl":"https://example.com"}`},
		{"sorts keys", `{"b":2,"a":1}`, `{"a":1,"b":2}`},
		{"invalid json returned unchanged", "not-json", "not-json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canonicalJSONString(tt.in); got != tt.want {
				t.Errorf("canonicalJSONString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRequireAtLeast(t *testing.T) {
	tests := []struct {
		name    string
		server  ServerVersion
		want    bool
		wantErr bool
	}{
		{"older v4", ServerVersion{Major: 4, Minor: 14}, false, true},
		{"older v5 minor", ServerVersion{Major: 5, Minor: 1}, false, true},
		{"exact", ServerVersion{Major: 5, Minor: 2}, true, false},
		{"newer minor", ServerVersion{Major: 5, Minor: 3}, true, false},
		{"newer major", ServerVersion{Major: 6, Minor: 0}, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := requireAtLeast(&Data{ServerVersion: tt.server}, 5, 2, "Feature", &diags)
			if got != tt.want {
				t.Errorf("requireAtLeast(%d.%d, 5, 2) = %v, want %v", tt.server.Major, tt.server.Minor, got, tt.want)
			}
			if diags.HasError() != tt.wantErr {
				t.Errorf("requireAtLeast(%d.%d, 5, 2) HasError = %v, want %v", tt.server.Major, tt.server.Minor, diags.HasError(), tt.wantErr)
			}
		})
	}
}

func TestServiceAccountName(t *testing.T) {
	tests := []struct {
		username string
		want     string
		wantOK   bool
	}{
		{"svc:ci", "ci", true},
		{"SVC:ci", "ci", true},
		{"Svc:", "", true},
		{"svc", "", false},
		{"admin", "", false},
		{"ci:svc:x", "", false},
	}

	for _, tt := range tests {
		got, ok := serviceAccountName(tt.username)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("serviceAccountName(%q) = (%q, %v), want (%q, %v)", tt.username, got, ok, tt.want, tt.wantOK)
		}
	}
}
