package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// parseCompositeID parses a composite ID in the format "part1/part2" and returns the two parts.
// The partNames are used for error messages to make them more descriptive.
func parseCompositeID(id string, part1Name, part2Name string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected format '%s/%s', got: %s", part1Name, part2Name, id)
	}

	return parts[0], parts[1], nil
}

// jsonStringsEquivalent reports whether a and b encode the same JSON value,
// ignoring formatting differences such as whitespace and key order. If either
// string is not valid JSON, it falls back to plain string comparison.
func jsonStringsEquivalent(a, b string) bool {
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return a == b
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return a == b
	}
	return reflect.DeepEqual(av, bv)
}

// canonicalJSONString re-serializes s into Go's canonical compact JSON form
// (no insignificant whitespace, object keys sorted), so that the same JSON
// value always produces the same string regardless of how the server chose to
// format it. If s is not valid JSON it is returned unchanged.
func canonicalJSONString(s string) string {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}

	b, err := json.Marshal(v)
	if err != nil {
		return s
	}
	return string(b)
}
