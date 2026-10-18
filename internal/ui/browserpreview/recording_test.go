package browserpreview

import (
	"context"
	"testing"
	"whaleisle/internal/desktop/browserrecording"
)

// No native window, Host, page capture or codec is constructed in this fixture.
func TestBrowserRecordingCancellationDropsArtifact(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tab := &Tab{}
	r := &browserRecording{tab: tab, ctx: ctx, cancel: cancel, phase: "ready", artifact: &browserrecording.Artifact{Data: []byte("synthetic")}}
	b := &Controller{Tabs: []*Tab{tab}, recording: r}
	b.CancelAction()
	if b.recording != nil || ctx.Err() != context.Canceled || r.artifact != nil {
		t.Fatal("cancel retained recorder or encoded bytes")
	}
	if tab.ActionMessage == "" {
		t.Fatal("cancel did not show discarded state")
	}
	b.CancelAction()
	// A later source/Host cannot make an old result eligible for export.
	if b.recordingCurrent(r) {
		t.Fatal("cancelled result remained exportable")
	}
}
