package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestMainWindowCaptionPlateSitsAtTop12 pins the native controls plate of the
// main window: 32x32 buttons 12 DIP into the 48 DIP caption band, the geometry the
// launcher draws (harness-chrome-inject.js:12,89,96-97; launcher.css:21,
// launcher_window.go). A centred pair would put the glyphs 4 DIP above the band's
// own title row, which starts at y=12.
func TestMainWindowCaptionPlateSitsAtTop12(t *testing.T) {
	body, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	plates := regexp.MustCompile(`WindowsCaptionButtons:\s*&mygo\.WindowsCaptionButtonOptions\{[^}]*\}`).FindAllString(source, -1)
	if len(plates) == 0 {
		t.Fatal("main window no longer declares a native caption plate")
	}
	for _, plate := range plates {
		if !strings.Contains(plate, "Top: 12") || !strings.Contains(plate, "Width: 32") || !strings.Contains(plate, "Height: 32") {
			t.Fatalf("caption plate %q is not the 32x32 top-12 plate", plate)
		}
	}
}

// TestSiblingCaptionPlatesShareTop12 keeps the other two custom-caption surfaces
// on the main window's plate. The welcome window (600x700, frame:false like
// welcome-window.js:36) still declared Top 8, which put its close glyph 4 DIP
// above the launcher's and the main window's; a single 12-DIP inset is the
// batch-B rule, so both must stay 32x32 at top 12.
func TestSiblingCaptionPlatesShareTop12(t *testing.T) {
	platePattern := regexp.MustCompile(`WindowsCaptionButtons:\s*&mygo\.WindowsCaptionButtonOptions\{[^}]*\}`)
	for _, file := range []string{"welcome_window.go", "launcher_window.go"} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		plates := platePattern.FindAllString(string(body), -1)
		if len(plates) == 0 {
			t.Fatalf("%s no longer declares a native caption plate", file)
		}
		for _, plate := range plates {
			if !strings.Contains(plate, "Top: 12") || !strings.Contains(plate, "Width: 32") || !strings.Contains(plate, "Height: 32") {
				t.Fatalf("%s caption plate %q is not the 32x32 top-12 plate", file, plate)
			}
		}
	}
}
