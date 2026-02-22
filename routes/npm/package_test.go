package routes

import (
	"testing"
	"time"
)

func TestSortedVersionTags(t *testing.T) {
	versions := map[string]NPMVersion{
		"1.0.0": {},
		"1.1.0": {},
		"2.0.0": {},
	}
	versionTimes := map[string]time.Time{
		"1.0.0": time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
		"1.1.0": time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC),
		"2.0.0": time.Date(2023, time.January, 3, 0, 0, 0, 0, time.UTC),
	}

	tags := sortedVersionTags(versions, versionTimes)
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(tags))
	}

	if tags[0].Version != "2.0.0" || tags[1].Version != "1.1.0" || tags[2].Version != "1.0.0" {
		t.Fatalf("unexpected version order: %#v", tags)
	}
}

func TestFormatRepoURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "git+https with git suffix",
			in:   "git+https://github.com/npm/cli.git",
			want: "https://github.com/npm/cli",
		},
		{
			name: "ssh github url",
			in:   "git+ssh://git@github.com/npm/cli.git",
			want: "https://github.com/npm/cli",
		},
		{
			name: "scp style github url",
			in:   "git@github.com:npm/cli.git",
			want: "https://github.com/npm/cli",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatRepoURL(tc.in)
			if got != tc.want {
				t.Fatalf("formatRepoURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
