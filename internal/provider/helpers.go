package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// secretNameRegex is the server-side pattern for secret names, from the
// create-secret-request schema of the /api/v2 OpenAPI spec.
var secretNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// serviceAccountNameRegex is the server-side pattern for service account
// names, from the service-account-name schema of the /api/v2 OpenAPI spec,
// minus its negative lookahead for the reserved "svc:" prefix (Go's regexp has
// no lookaheads), which serviceAccountNameValidator checks instead.
var serviceAccountNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9+=,.:@_-]*$`)

// serviceAccountUsernamePrefix is the reserved username prefix of Dependency-Track
// service accounts (5.2+). The server matches it case-insensitively.
const serviceAccountUsernamePrefix = "svc:"

// serviceAccountName returns the service account name for a username that
// carries the reserved "svc:" prefix, and whether it does.
func serviceAccountName(username string) (string, bool) {
	if len(username) < len(serviceAccountUsernamePrefix) ||
		!strings.EqualFold(username[:len(serviceAccountUsernamePrefix)], serviceAccountUsernamePrefix) {
		return "", false
	}
	return username[len(serviceAccountUsernamePrefix):], true
}

// serviceAccountNameValidator rejects service account names that start with
// the reserved "svc:" prefix: the name is given without it.
type serviceAccountNameValidator struct{}

func (v serviceAccountNameValidator) Description(ctx context.Context) string {
	return "must not start with the reserved \"svc:\" prefix"
}

func (v serviceAccountNameValidator) MarkdownDescription(ctx context.Context) string {
	return "must not start with the reserved `svc:` prefix"
}

func (v serviceAccountNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, ok := serviceAccountName(req.ConfigValue.ValueString()); ok {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Service Account Name",
			"The service account name must not include the reserved \"svc:\" prefix; Dependency-Track adds it to form the username.")
	}
}

// requireV5 verifies that the configured server is Dependency-Track v5 or
// newer. Resources built on the /api/v2 API (which does not exist on v4) call
// this at the top of every CRUD method; it appends an actionable error and
// returns false when the server is too old.
func requireV5(data *Data, resourceName string, diags *diag.Diagnostics) bool {
	if data.IsV5() {
		return true
	}

	diags.AddError(
		"Dependency-Track v5 Required",
		fmt.Sprintf("%s uses the /api/v2 API, which is only available on Dependency-Track v5 and newer. "+
			"The configured server reports version %d.%d.",
			resourceName, data.ServerVersion.Major, data.ServerVersion.Minor),
	)
	return false
}

// requireAtLeast verifies that the configured server is Dependency-Track
// major.minor or newer. Features introduced in a specific v5 minor release
// (e.g. 5.2 service accounts) call this; it appends an actionable error and
// returns false when the server is too old.
//
//nolint:unparam // major keeps the gate general; all current gates are 5.x minors
func requireAtLeast(data *Data, major, minor int, feature string, diags *diag.Diagnostics) bool {
	if data.ServerVersion.AtLeast(major, minor) {
		return true
	}

	diags.AddError(
		fmt.Sprintf("Dependency-Track %d.%d Required", major, minor),
		fmt.Sprintf("%s requires Dependency-Track %d.%d or newer. The configured server reports version %d.%d.",
			feature, major, minor, data.ServerVersion.Major, data.ServerVersion.Minor),
	)
	return false
}

// parseCompositeID parses a composite ID in the format "part1/part2" and returns the two parts.
// The partNames are used for error messages to make them more descriptive.
func parseCompositeID(id string, part1Name, part2Name string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected format '%s/%s', got: %s", part1Name, part2Name, id)
	}

	return parts[0], parts[1], nil
}

// parseCompositeID3 parses a composite ID in the format "part1/part2/part3" and
// returns the three parts. The partNames are used for error messages to make
// them more descriptive.
func parseCompositeID3(id string, part1Name, part2Name, part3Name string) (string, string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("expected format '%s/%s/%s', got: %s", part1Name, part2Name, part3Name, id)
	}

	return parts[0], parts[1], parts[2], nil
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
