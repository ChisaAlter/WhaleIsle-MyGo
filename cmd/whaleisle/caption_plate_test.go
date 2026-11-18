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

// First-run onboarding retains the same native caption geometry as the workbench.
func TestWelcomeCaptionPlateSharesTop12(t *testing.T) {
	platePattern := regexp.MustCompile(`WindowsCaptionButtons:\s*&mygo\.WindowsCaptionButtonOptions\{[^}]*\}`)
	for _, file := range []string{"welcome_window.go"} {
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
