package hostterminal

import (
	"strings"
	"testing"
	"time"
)

func TestDuplexProviderAckAndCloseUnblocksPendingWrite(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 50)
	r.mu.Lock()
	r.noAck = true
	r.mu.Unlock()
	done := make(chan error, 1)
	go func() { _, err := a.Write([]byte("x")); done <- err }()
	deadline := time.Now().Add(time.Second)
	for r.count("input/write") == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if r.count("input/write") != 1 {
		t.Fatal("command was not transmitted")
	}
	select {
	case <-done:
		t.Fatal("write completed before provider acknowledgement")
	default:
	}
	s.Show("other", true)
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("detached pending command reported success")
		}
	case <-time.After(time.Second):
		t.Fatal("detach retained pending writer")
	}
	if r.count("input/write") != 1 || r.count("terminal/write") != 0 {
		t.Fatal("command replayed or used old RPC")
	}
}

func TestDuplexEscapedInputRespectsCarrierWindow(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 0)
	s.mu.Lock()
	a.inputWindow = &inputWindow{MaxBytes: 70, FrameOverheadBytes: 20}
	s.mu.Unlock()
	if _, err := a.Write([]byte(strings.Repeat("\x00", 8))); err == nil {
		t.Fatal("escaped JSON command exceeded advertised carrier window")
	}
	if r.count("input/write") != 0 {
		t.Fatal("oversized encoded frame was sent")
	}
}

func TestDuplexInputAckHasIndependentStrictSequence(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 500)
	if _, err := a.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := a.Resize(90, 30); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	sequence := float64(0)
	for _, call := range r.calls {
		if strings.HasPrefix(call.method, "input/") {
			sequence++
			if call.args["sequence"] != sequence {
				t.Fatal("output sequence polluted command ordering", call)
			}
		}
	}
	if sequence != 2 {
		t.Fatal("missing duplex commands")
	}
}

func TestRetentionEndDuringExplicitClosePreservesClosingState(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	hold := nextOpen(t, r, "terminal/retain")
	hold.stream.send(map[string]any{"type": "retained"})
	_ = nextOpen(t, r, "terminal/attach")
	s.mu.Lock()
	e := s.entries[terminalKey{"session", "term"}]
	e.closing = true
	e.phase = "closing"
	completed := make(chan struct{}, 1)
	s.notify = func() {
		select {
		case completed <- struct{}{}:
		default:
		}
	}
	s.mu.Unlock()
	hold.stream.Cancel()
	// Observe the completion callback before the close RPC's result.
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("retention completion was not observed")
	}
	v := s.Snapshot("session")
	if v.Phase != "closing" || v.Error != "" {
		t.Fatalf("expected retention end replaced explicit close: %+v", v)
	}
}
