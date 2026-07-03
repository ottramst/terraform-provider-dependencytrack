package provider

import (
	"fmt"
	"strconv"
	"strings"
)

// ServerVersion is a parsed Dependency-Track server version, as reported by the
// unauthenticated GET /api/version endpoint (client-go's About.Get).
type ServerVersion struct {
	Raw   string
	Major int
	Minor int
}

// parseServerVersion parses a Dependency-Track version string such as "4.14.2",
// "5.0.2", or "5.0.0-SNAPSHOT" into a ServerVersion.
//
// Any build/pre-release suffix (everything from the first '-' onward) is
// stripped before parsing the numeric components; it is preserved verbatim in
// the returned Raw field. The patch component, if present, is parsed but
// otherwise ignored.
//
// Design choice: a bare major version with no dot at all (e.g. "5") is
// accepted with an implied minor of 0 rather than rejected, since it is a
// plausible (if unusual) version string and there is no ambiguity in how to
// interpret it. Once a dot is present, however, both the major and minor
// components must parse as integers or the input is rejected.
func parseServerVersion(raw string) (ServerVersion, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ServerVersion{}, fmt.Errorf("parseServerVersion: version string is empty")
	}

	numeric := trimmed
	if idx := strings.IndexByte(numeric, '-'); idx >= 0 {
		numeric = numeric[:idx]
	}

	parts := strings.Split(numeric, ".")

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ServerVersion{}, fmt.Errorf("parseServerVersion: invalid major version in %q: %w", raw, err)
	}

	minor := 0
	if len(parts) >= 2 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return ServerVersion{}, fmt.Errorf("parseServerVersion: invalid minor version in %q: %w", raw, err)
		}
	}

	return ServerVersion{Raw: trimmed, Major: major, Minor: minor}, nil
}

// IsV5 reports whether the server is running Dependency-Track 5.x or newer.
func (v ServerVersion) IsV5() bool {
	return v.Major >= 5
}

// AtLeast reports whether the server version is greater than or equal to major.minor.
func (v ServerVersion) AtLeast(major, minor int) bool {
	if v.Major != major {
		return v.Major > major
	}
	return v.Minor >= minor
}

// String returns the original, unparsed version string.
func (v ServerVersion) String() string {
	return v.Raw
}
