package pluginui

import (
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestMirrorCompositionDefersCaptureUntilInputIdleAndNoMarkedText(t *testing.T) {
	state := &mirrorTextState{clients: map[string]*MirrorTextClient{}}
	if state.captureDeferred() {
		t.Fatal("idle original source capture stayed blocked")
	}
	state.lastInput = time.Now()
	if !state.captureDeferred() {
		t.Fatal("capture overtook acknowledged text input")
	}
	state.lastInput = time.Now().Add(-2 * time.Second)
	state.clients["A"] = &MirrorTextClient{marked: &ui.TextInputRange{Start: 1, End: 3}}
	if !state.captureDeferred() {
		t.Fatal("capture overtook native composition interval")
	}
	state.clients["A"].marked = nil
	if state.captureDeferred() {
		t.Fatal("committed idle source capture stayed blocked")
	}
	state.queue = []mirrorTextJob{{event: mygo.BrowserTextEvent{Kind: mygo.BrowserTextComposition}}}
	if !state.captureDeferred() {
		t.Fatal("queued preedit lost source priority")
	}
}
