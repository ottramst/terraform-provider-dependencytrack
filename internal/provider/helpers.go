package provider

import (
	"encoding/json"
	"fmt"
	"io"
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

// decodeJSON decodes JSON from an io.Reader into the provided result interface.
func decodeJSON(reader io.Reader, result interface{}) error {
	return json.NewDecoder(reader).Decode(result)
}
