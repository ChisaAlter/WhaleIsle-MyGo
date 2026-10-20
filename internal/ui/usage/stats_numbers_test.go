package usage

import "testing"

// TestFormatCompactTokens pins the dashboard's number format: counts stay
// readable at a glance instead of printing every digit.
func TestFormatCompactTokens(t *testing.T) {
	cases := []struct {
		value float64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{12400, "12.4K"},
		{999999, "1000K"},
		{12400000, "12.4M"},
		{77700000, "77.7M"},
		{1200000000, "1.2B"},
	}
	for _, tc := range cases {
		if got := formatCompactTokens(tc.value); got != tc.want {
			t.Fatalf("formatCompactTokens(%v) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
