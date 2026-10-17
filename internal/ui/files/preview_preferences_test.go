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
