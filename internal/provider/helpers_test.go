package provider

import "testing"

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
