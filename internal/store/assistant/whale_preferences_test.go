package assistant

import (
	"errors"
	"testing"

	"whaleisle/internal/desktop/pluginrpc"
)

func TestWhalePreferencesWriteFailureKeepsPageInteractiveAndRollsBack(t *testing.T) {
	ed := &WhalePreferencesEditor{}
	const host = "http://127.0.0.1:4321"
	ed.Bind(host)

	// Initial load succeeds with confirmed server state.
	loadTicket, ok := ed.Begin(host, false)
	if !ok {
		t.Fatal("Begin(load) failed")
	}
	serverPrefs := pluginrpc.WhalePreferences{
		HomeDir:              "/whale",
		SessionID:            "session-whale-1",
		ModelProvider:        "deepseek",
		ModelModel:           "deepseek-chat",
		ModelReasoningEffort: "medium",
		IMDefault:            true,
	}
	ed.Finish(loadTicket, serverPrefs, nil, nil, nil, nil)
	if snap := ed.Snapshot().Value; !snap.Loaded || snap.Route != serverPrefs.Route() || !snap.IMDefault {
		t.Fatalf("unexpected initial loaded snapshot: %+v", snap)
	}

	// 1) Model/effort write rejected by dsh-whale: must stay Loaded=true, roll back
	// Route to confirmed server value, surface Error, and allow immediate retry.
	rejectedRoute := pluginrpc.WhaleModelRoute{Provider: "deepseek", Model: "deepseek-chat", ReasoningEffort: "low"}
	ed.Edit(host, rejectedRoute, true)
	writeTicket, ok := ed.Begin(host, true)
	if !ok {
		t.Fatal("Begin(model write) failed")
	}
	ed.Finish(writeTicket, pluginrpc.WhalePreferences{}, nil, nil, &WhalePreferencesCommand{
		Action: "model",
		Route:  rejectedRoute,
	}, errors.New("effort low requires explicit model"))

	afterModelErr := ed.Snapshot().Value
	if !afterModelErr.Loaded {
		t.Fatal("model write failure locked page by clearing Loaded=false")
	}
	if afterModelErr.Error == nil {
		t.Fatal("model write failure did not record Error")
	}
	if afterModelErr.Route != serverPrefs.Route() {
		t.Fatalf("rejected Route was not rolled back: got %+v want %+v", afterModelErr.Route, serverPrefs.Route())
	}

	// 2) Immediately editing and saving IM default must still work without reload;
	// a rejected IM write must roll back IMDefault while staying Loaded=true.
	ed.Edit(host, serverPrefs.Route(), false)
	if snap := ed.Snapshot().Value; snap.IMDefault != false {
		t.Fatal("Edit after failed write was blocked")
	}
	imTicket, ok := ed.Begin(host, true)
	if !ok {
		t.Fatal("Begin(im write) blocked after previous write failure")
	}
	ed.Finish(imTicket, pluginrpc.WhalePreferences{}, nil, nil, &WhalePreferencesCommand{
		Action:    "im",
		IMDefault: false,
	}, errors.New("im save rejected"))
	afterIMErr := ed.Snapshot().Value
	if !afterIMErr.Loaded || afterIMErr.IMDefault != true || afterIMErr.Error == nil {
		t.Fatalf("IM write failure state mismatch: loaded=%v im=%v err=%v", afterIMErr.Loaded, afterIMErr.IMDefault, afterIMErr.Error)
	}

	// 3) An initial load failure before Loaded=true still leaves Loaded=false.
	ed2 := &WhalePreferencesEditor{}
	ed2.Bind(host)
	t2, ok := ed2.Begin(host, false)
	if !ok {
		t.Fatal("Begin(initial load) failed")
	}
	ed2.Finish(t2, pluginrpc.WhalePreferences{}, nil, nil, nil, errors.New("host down"))
	if snap := ed2.Snapshot().Value; snap.Loaded || snap.Error == nil {
		t.Fatalf("initial load failure should keep Loaded=false and set Error: %+v", snap)
	}
}
