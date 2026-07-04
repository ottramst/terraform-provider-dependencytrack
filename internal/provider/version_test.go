package provider

import "testing"

func TestParseServerVersion(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantMajor   int
		wantMinor   int
		wantRaw     string
		wantErr     bool
		description string
	}{
		{
			name:      "full semver",
			raw:       "4.14.2",
			wantMajor: 4,
			wantMinor: 14,
			wantRaw:   "4.14.2",
		},
		{
			name:      "v5 full semver",
			raw:       "5.0.2",
			wantMajor: 5,
			wantMinor: 0,
			wantRaw:   "5.0.2",
		},
		{
			name:      "pre-release suffix stripped for parsing but kept in Raw",
			raw:       "5.0.0-SNAPSHOT",
			wantMajor: 5,
			wantMinor: 0,
			wantRaw:   "5.0.0-SNAPSHOT",
		},
		{
			name:      "major.minor only",
			raw:       "4.13",
			wantMajor: 4,
			wantMinor: 13,
			wantRaw:   "4.13",
		},
		{
			name:      "surrounding whitespace trimmed",
			raw:       "  4.14.2  ",
			wantMajor: 4,
			wantMinor: 14,
			wantRaw:   "4.14.2",
		},
		{
			name:    "empty string is an error",
			raw:     "",
			wantErr: true,
		},
		{
			name:    "whitespace only is an error",
			raw:     "   ",
			wantErr: true,
		},
		{
			name:    "garbage input is an error",
			raw:     "abc",
			wantErr: true,
		},
		{
			name:        "major-only version is accepted with an implied minor of 0",
			description: "Design choice: a bare major version like \"5\" (no dot) is treated as {Major: 5, Minor: 0} rather than an error, since Dependency-Track's own /api/version has historically been well-formed but we don't want a future major-only tag to hard-fail version detection.",
			raw:         "5",
			wantMajor:   5,
			wantMinor:   0,
			wantRaw:     "5",
		},
		{
			name:    "unparseable minor is an error",
			raw:     "4.abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseServerVersion(tt.raw)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseServerVersion(%q) = %+v, want error", tt.raw, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseServerVersion(%q) returned unexpected error: %s", tt.raw, err)
			}

			if got.Major != tt.wantMajor {
				t.Errorf("parseServerVersion(%q).Major = %d, want %d", tt.raw, got.Major, tt.wantMajor)
			}
			if got.Minor != tt.wantMinor {
				t.Errorf("parseServerVersion(%q).Minor = %d, want %d", tt.raw, got.Minor, tt.wantMinor)
			}
			if got.Raw != tt.wantRaw {
				t.Errorf("parseServerVersion(%q).Raw = %q, want %q", tt.raw, got.Raw, tt.wantRaw)
			}
		})
	}
}

func TestServerVersionIsV5(t *testing.T) {
	tests := []struct {
		version ServerVersion
		want    bool
	}{
		{version: ServerVersion{Major: 4, Minor: 14}, want: false},
		{version: ServerVersion{Major: 5, Minor: 0}, want: true},
		{version: ServerVersion{Major: 6, Minor: 0}, want: true},
	}

	for _, tt := range tests {
		if got := tt.version.IsV5(); got != tt.want {
			t.Errorf("ServerVersion{Major: %d}.IsV5() = %v, want %v", tt.version.Major, got, tt.want)
		}
	}
}

func TestServerVersionAtLeast(t *testing.T) {
	tests := []struct {
		name    string
		version ServerVersion
		major   int
		minor   int
		want    bool
	}{
		{name: "equal", version: ServerVersion{Major: 4, Minor: 14}, major: 4, minor: 14, want: true},
		{name: "higher minor", version: ServerVersion{Major: 4, Minor: 15}, major: 4, minor: 14, want: true},
		{name: "lower minor", version: ServerVersion{Major: 4, Minor: 13}, major: 4, minor: 14, want: false},
		{name: "higher major beats lower minor requirement", version: ServerVersion{Major: 5, Minor: 0}, major: 4, minor: 14, want: true},
		{name: "lower major", version: ServerVersion{Major: 4, Minor: 14}, major: 5, minor: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.AtLeast(tt.major, tt.minor); got != tt.want {
				t.Errorf("ServerVersion{%d.%d}.AtLeast(%d, %d) = %v, want %v", tt.version.Major, tt.version.Minor, tt.major, tt.minor, got, tt.want)
			}
		})
	}
}

func TestServerVersionString(t *testing.T) {
	v := ServerVersion{Raw: "5.0.0-SNAPSHOT", Major: 5, Minor: 0}
	if got := v.String(); got != "5.0.0-SNAPSHOT" {
		t.Errorf("ServerVersion.String() = %q, want %q", got, "5.0.0-SNAPSHOT")
	}
}
