package frame

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
)

// Exercise the actual per-frame bridge, excluding unfinished stream fragments
// and ensuring it uses only the selected session's settled start records.
func TestApprovalContextFrameBridge(t *testing.T) {
	store := interaction.New()
	defer store.Detach()
	ready := make(chan struct{}, 1)
	store.Attach(func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = store.HandleWaterfall(dsh.EventsWaterfall{AgentID: "A", EventID: "approval", Event: interaction.EventApproval, Request: json.RawMessage(`{"toolName":"exec_command","callId":"call"}`)})
	}()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("pending approval not opened")
	}
	events := []session.Message{
		{Streaming: true, Raw: json.RawMessage(`{"type":"tool/call","data":{"callId":"call","arguments":{"command":"unfinished"}}}`)},
		{Raw: json.RawMessage(`{"type":"tool/call","data":{"callId":"call","arguments":{"command":"original"}}}`)},
	}
	syncQuestionHistory(store, session.ViewState{SelectedID: "B", Messages: events})
	if store.Snapshot().Pending[0].Args != "" {
		t.Fatal("foreign selected session supplied arguments")
	}
	syncQuestionHistory(store, session.ViewState{SelectedID: "A", Messages: events})
	if got := store.Snapshot().Pending[0].Args; !strings.Contains(got, "original") || strings.Contains(got, "unfinished") {
		t.Fatalf("arguments: %q", got)
	}
	store.ResolveDeny("approval")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("approval stuck")
	}
}
