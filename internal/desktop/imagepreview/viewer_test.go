package imagepreview

import (
	"fmt"
	"strings"
	"testing"
)

func TestB23InertImageZoomControls(t *testing.T) {
	markup := imageViewer("example.gif", "data:image/gif;base64,R0lGODlh")
	for _, want := range []string{`id="zoom-fit"`, `checked`, `overflow:auto`, `max-width:100%`, `zoom:1`, `tabindex="0"`, `referrerpolicy="no-referrer"`, `draggable="false"`} {
		if !strings.Contains(markup, want) {
			t.Fatalf("missing %s", want)
		}
	}
	for percent := 25; percent <= 400; percent += 25 {
		for _, want := range []string{fmt.Sprintf(`id="zoom-%d"`, percent), fmt.Sprintf(`#zoom-%d:checked~main img{zoom:%g}`, percent, float64(percent)/100)} {
			if !strings.Contains(markup, want) {
				t.Fatalf("missing zoom contract %s", want)
			}
		}
	}
	for _, bad := range []string{"<script", "onclick=", "onload=", "<form", "https://", "fetch(", "postMessage"} {
		if strings.Contains(markup, bad) {
			t.Fatalf("inert image acquired %q", bad)
		}
	}
	if strings.Count(markup, `src="data:image/gif;base64,R0lGODlh"`) != 1 {
		t.Fatal("image bytes were not kept in one inert image")
	}
}
