package recovery

import (
	"encoding/json"
	"testing"
)

func TestTranscriptWidthLayoutRoundTrip(t *testing.T) {
	original := Layout{TranscriptWidth: 960, Sidebar: 280, Right: 450}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored Layout
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.TranscriptWidth != 960 || restored.Sidebar != 280 || restored.Right != 450 {
		t.Fatal("lost independent width preference")
	}
	var legacy Layout
	if err = json.Unmarshal([]byte(`{"sidebar":280}`), &legacy); err != nil || legacy.TranscriptWidth != 0 {
		t.Fatal("legacy layout must use default adaptive width")
	}
}
