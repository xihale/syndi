package routeutils

import "testing"

func TestParsePositiveInt(t *testing.T) {
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
			got := ParsePositiveInt(tc.raw, tc.defaultValue, tc.maxValue)
			if got != tc.want {
				t.Fatalf("ParsePositiveInt(%q, %d, %d) = %d, want %d", tc.raw, tc.defaultValue, tc.maxValue, got, tc.want)
			}
		})
	}
}

func TestParseOptionalPositiveInt(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want *int
	}{
		{name: "empty returns nil", raw: "", want: nil},
		{name: "invalid returns nil", raw: "abc", want: nil},
		{name: "non-positive returns nil", raw: "0", want: nil},
		{name: "positive returns pointer", raw: "42", want: intPtr(42)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseOptionalPositiveInt(tc.raw)
			if tc.want == nil && got != nil {
				t.Fatalf("ParseOptionalPositiveInt(%q) = %v, want nil", tc.raw, *got)
			}
			if tc.want != nil {
				if got == nil || *got != *tc.want {
					t.Fatalf("ParseOptionalPositiveInt(%q) = %v, want %v", tc.raw, derefInt(got), *tc.want)
				}
			}
		})
	}
}

func TestParseBool(t *testing.T) {
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
		{name: "accepts mixed case", raw: "YeS", defaultValue: false, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseBool(tc.raw, tc.defaultValue)
			if got != tc.want {
				t.Fatalf("ParseBool(%q, %t) = %t, want %t", tc.raw, tc.defaultValue, got, tc.want)
			}
		})
	}
}

func TestParseEnum(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		defaultValue string
		allowed      []string
		want         string
	}{
		{
			name:         "returns canonical matched value",
			raw:          "TOP",
			defaultValue: "hot",
			allowed:      []string{"hot", "new", "top"},
			want:         "top",
		},
		{
			name:         "returns default when invalid",
			raw:          "unknown",
			defaultValue: "hot",
			allowed:      []string{"hot", "new", "top"},
			want:         "hot",
		},
		{
			name:         "falls back to first allowed when default invalid",
			raw:          "unknown",
			defaultValue: "invalid-default",
			allowed:      []string{"a", "b"},
			want:         "a",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseEnum(tc.raw, tc.defaultValue, tc.allowed...)
			if got != tc.want {
				t.Fatalf("ParseEnum(%q, %q, %v) = %q, want %q", tc.raw, tc.defaultValue, tc.allowed, got, tc.want)
			}
		})
	}
}

func intPtr(v int) *int {
	return &v
}

func derefInt(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}
