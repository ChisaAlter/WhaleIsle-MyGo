package hostterminal

import (
	"errors"
	"testing"
	"time"
)

// One-shot control calls: write and resize are unary RPCs bound to the fresh
// attachment identity, never uplink items on the follow stream.
func TestControlCallsBindAttachmentIdentity(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 3)
	if _, err := a.Write([]byte("ls\r")); err != nil {
		t.Fatal(err)
	}
	if err := a.Resize(100, 40); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var writes, resizes int
	for _, c := range r.calls {
		switch c.method {
		case "terminal/write":
			writes++
			if c.args["agentId"] != "session" || c.args["id"] != "term" || c.args["attachmentId"] != a.ID || c.args["data"] != "ls\r" {
				t.Fatal(c)
			}
		case "terminal/resize":
			resizes++
			if c.args["agentId"] != "session" || c.args["id"] != "term" || c.args["attachmentId"] != a.ID || c.args["cols"] != float64(100) || c.args["rows"] != float64(40) {
				t.Fatal(c)
			}
		case "input/write", "input/resize":
			t.Fatal("obsolete duplex command transmitted", c)
		}
	}
	if writes != 1 || resizes != 1 {
		t.Fatal("one-shot control calls missing", writes, resizes)
	}
}

// A control call with an uncertain outcome fails exactly that attachment; the
// input is never replayed and the attachment never accepts further input.
func TestControlFailureFailsAttachmentWithoutReplay(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 7)
	r.mu.Lock()
	r.writeErr = errors.New("response lost")
	r.mu.Unlock()
	if _, err := a.Write([]byte("x")); err == nil {
		t.Fatal("missing failure")
	}
	if r.count("terminal/write") != 1 {
		t.Fatal("uncertain write not transmitted once")
	}
	if _, err := a.Write([]byte("y")); err == nil {
		t.Fatal("failed attachment remained writable")
	}
	if r.count("terminal/write") != 1 {
		t.Fatal("input replayed after failure")
	}
	if s.Snapshot("session").Phase != "failed" {
		t.Fatal("failed control call did not surface")
	}
}

func TestRetentionEndDuringExplicitClosePreservesClosingState(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	hold := nextOpen(t, r, "terminal/retain")
	hold.stream.send(map[string]any{"type": "retained"})
	_ = nextOpen(t, r, "terminal/follow")
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
