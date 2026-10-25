package theme

import "testing"

// TestScrollbarColorsMatchTheHarness pins the scroll bar pairs to the tokens
// they were sampled from (ui-theme/src/styles/design-platform.css): the light
// half resolves both levels to neutral-200/300, the dark half takes neutral-700
// and -600 for a page and neutral-600 and -550 for a raised surface.
func TestScrollbarColorsMatchTheHarness(t *testing.T) {
	if rest, hover := ScrollbarColors(false); rest != "rgb(229, 229, 229)" || hover != "rgb(212, 212, 212)" {
		t.Fatalf("light scrollbar = %q / %q, want neutral-200 / neutral-300", rest, hover)
	}
	if rest, hover := ScrollbarColors(true); rest != "rgb(60, 60, 61)" || hover != "rgb(84, 85, 87)" {
		t.Fatalf("dark scrollbar = %q / %q, want neutral-700 / neutral-600", rest, hover)
	}
	// The raised (level-2) pair of the dark half, which the reference rebinds on a
	// menu or a modal; the light half's two levels coincide.
	if ScrollbarRestDarkRaised != "rgb(84, 85, 87)" || ScrollbarHoverDarkRaised != "rgb(101, 103, 107)" {
		t.Fatalf("raised dark scrollbar = %q / %q, want neutral-600 / neutral-550", ScrollbarRestDarkRaised, ScrollbarHoverDarkRaised)
	}
	// Every pair has to be a color the shell can paint, not an unresolved token.
	for _, c := range []string{ScrollbarRestLight, ScrollbarHoverLight, ScrollbarRestDark, ScrollbarHoverDark, ScrollbarRestDarkRaised, ScrollbarHoverDarkRaised} {
		if _, ok := parseTokenColor(c); !ok {
			t.Fatalf("%q is not a color", c)
		}
	}
}
