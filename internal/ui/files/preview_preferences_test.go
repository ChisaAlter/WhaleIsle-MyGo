package files

import (
	"encoding/json"
	"testing"
)

func TestPreviewWrapPreferenceRoundTrip(t *testing.T) {
	p := PreviewPreferences{}
	if p.WordWrap {
		t.Fatal("original default is unwrapped")
	}
	p.WordWrap = true
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var restored PreviewPreferences
	if json.Unmarshal(raw, &restored) != nil || !restored.WordWrap {
		t.Fatal("wrap setting lost")
	}
}
func TestSourceAreaWidth(t *testing.T) {
	for _, tc := range []struct {
		wrap                    bool
		viewport, content, want float32
	}{
		{true, 400, 1200, 400}, {false, 400, 1200, 1216}, {false, 400, 80, 400}, {false, 0, 0, 1},
	} {
		if got := sourceAreaWidth(tc.wrap, tc.viewport, tc.content); got != tc.want {
			t.Errorf("width: %v want %v", got, tc.want)
		}
	}
}
func TestFloatingSnapshotKeepsDraftAndErrors(t *testing.T) {
	s, err := makePreviewSnapshot("/remote/root", "a.md", "disk", "unsaved", "changed on disk", false)
	if err != nil || s.Text != "unsaved" || !s.Dirty || s.Root != "/remote/root" {
		t.Fatalf("draft snapshot = %+v %v", s, err)
	}
	if _, err = makePreviewSnapshot("/remote/root", "a.md", "", "", "not loaded", false); err == nil {
		t.Fatal("failed preview opened")
	}
	s, err = makePreviewSnapshot("/remote/root", "a.svg", "disk", "unsaved", "", true)
	if err != nil || s.Image {
		t.Fatal("SVG draft was hidden by image mode")
	}
	if _, err = makePreviewSnapshot("", "a.md", "", "", "", false); err == nil {
		t.Fatal("unowned snapshot accepted")
	}
}
