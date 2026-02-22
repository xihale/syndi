package routes

import "testing"

func TestParsePositiveLimit(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		defaultValue int
		maxValue     int
		want         int
	}{
		{name: "uses default for empty", raw: "", defaultValue: 20, maxValue: 100, want: 20},
		{name: "uses parsed value", raw: "10", defaultValue: 20, maxValue: 100, want: 10},
		{name: "clamps to max", raw: "500", defaultValue: 20, maxValue: 100, want: 100},
		{name: "invalid falls back", raw: "abc", defaultValue: 20, maxValue: 100, want: 20},
		{name: "negative falls back", raw: "-1", defaultValue: 20, maxValue: 100, want: 20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePositiveLimit(tc.raw, tc.defaultValue, tc.maxValue)
			if got != tc.want {
				t.Fatalf("parsePositiveLimit(%q, %d, %d) = %d, want %d", tc.raw, tc.defaultValue, tc.maxValue, got, tc.want)
			}
		})
	}
}

func TestParseBoolDefault(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		defaultValue bool
		want         bool
	}{
		{name: "empty uses default", raw: "", defaultValue: true, want: true},
		{name: "true value", raw: "true", defaultValue: false, want: true},
		{name: "false value", raw: "no", defaultValue: true, want: false},
		{name: "invalid uses default", raw: "maybe", defaultValue: false, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseBoolDefault(tc.raw, tc.defaultValue)
			if got != tc.want {
				t.Fatalf("parseBoolDefault(%q, %t) = %t, want %t", tc.raw, tc.defaultValue, got, tc.want)
			}
		})
	}
}
