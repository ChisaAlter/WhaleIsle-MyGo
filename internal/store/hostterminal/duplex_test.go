package hostterminal

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestControlWaitsForProviderAckOnPhysicalStream(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 70)
	r.mu.Lock()
	r.delayAck = true
	r.mu.Unlock()
	done := make(chan error, 1)
	go func() { _, e := a.Write([]byte("abc")); done <- e }()
	wait(t, s, "session", func(View) bool { return r.count("input/write") == 1 })
	select {
	case e := <-done:
		t.Fatal("write returned before provider ack", e)
	default:
	}
	f := a.stream.(*fixtureStream)
	f.send(map[string]any{"type": "ack", "sequence": 1})
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("provider ack did not complete write")
	}
	go func() { done <- a.Resize(100, 40) }()
	wait(t, s, "session", func(View) bool { return r.count("input/resize") == 1 })
	select {
	case e := <-done:
		t.Fatal("resize returned before provider ack", e)
	default:
	}
	f.send(map[string]any{"type": "ack", "sequence": 2})
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("provider ack did not complete resize")
	}
}

func TestControlRefusalKeepsOutputWithoutReplayingInput(t *testing.T) {
	for _, reason := range []string{"read-only", "not-running"} {
		t.Run(reason, func(t *testing.T) {
			s, r := prepare(t)
			defer s.bind(nil, nil, false, nil)
			a := attach(t, s, r, 4)
			r.mu.Lock()
			r.refusal = reason
			r.mu.Unlock()
			_, err := a.Write([]byte("abc"))
			var refusal *controlRefusal
			if !errors.As(err, &refusal) || refusal.reason != reason {
				t.Fatalf("refusal=%v", err)
			}
			v := s.Snapshot("session")
			if v.Writable || v.Error != "" || v.Attachment != a {
				t.Fatalf("confirmed refusal lost output: %+v", v)
			}
			a.stream.(*fixtureStream).send(map[string]any{"type": "output", "sequence": 5, "data": "still receiving"})
			v = wait(t, s, "session", func(v View) bool { return v.Render != nil })
			if v.Render.Data != "still receiving" {
				t.Fatal(v.Render)
			}
			s.Acknowledge(a, v.Render.Revision)
			if _, e := a.Write([]byte("x")); e == nil || r.count("input/write") != 1 {
				t.Fatal("read-only command replayed", e)
			}
		})
	}
}

func TestEncodedInputWindowUsesWholeCommandAndEnvelope(t *testing.T) {
	for _, fits := range []bool{false, true} {
		t.Run(map[bool]string{false: "over limit", true: "exact boundary"}[fits], func(t *testing.T) {
			s, r := prepare(t)
			defer s.bind(nil, nil, false, nil)
			a := attach(t, s, r, 0)
			encoded, _ := json.Marshal(inputCommand{Type: "write", Sequence: 1, Data: "x"})
			capacity := int64(len(encoded) + 17)
			if !fits {
				capacity--
			}
			s.mu.Lock()
			a.inputWindow = &inputWindow{MaxBytes: capacity, FrameOverheadBytes: 17}
			s.mu.Unlock()
			_, err := a.Write([]byte("x"))
			if fits {
				if err != nil || r.count("input/write") != 1 {
					t.Fatal("exact encoded window refused", err)
				}
			} else if err == nil || r.count("input/write") != 0 || s.Snapshot("session").Phase != "failed" {
				t.Fatal("oversized encoded command transmitted", err)
			}
		})
	}
}

func TestWrongProviderAckFailsAttachmentAndNeverReplays(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 10)
	r.mu.Lock()
	r.delayAck = true
	r.mu.Unlock()
	done := make(chan error, 1)
	go func() { _, e := a.Write([]byte("x")); done <- e }()
	wait(t, s, "session", func(View) bool { return r.count("input/write") == 1 })
	a.stream.(*fixtureStream).send(map[string]any{"type": "ack", "sequence": 2})
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("wrong ack succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("invalid ack did not release writer")
	}
	if s.Snapshot("session").Phase != "failed" {
		t.Fatal("invalid ack attachment not failed")
	}
	if _, e := a.Write([]byte("y")); e == nil || r.count("input/write") != 1 {
		t.Fatal("unknown input replayed", e)
	}
}

// Control is ordered on one physical attach stream, with provider ack before
// success. No separate control RPC can escape that attachment's generation.
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
		case "input/write":
			writes++
			if c.args["sequence"] != float64(1) || c.args["data"] != "ls\r" {
				t.Fatal(c)
			}
		case "input/resize":
			resizes++
			if c.args["sequence"] != float64(2) || c.args["cols"] != float64(100) || c.args["rows"] != float64(40) {
				t.Fatal(c)
			}
		case "terminal/write", "terminal/resize":
			t.Fatal("obsolete unary command transmitted", c)
		}
	}
	if writes != 1 || resizes != 1 {
		t.Fatal("ordered attach commands missing", writes, resizes)
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
	if r.count("input/write") != 1 {
		t.Fatal("uncertain write not transmitted once")
	}
	if _, err := a.Write([]byte("y")); err == nil {
		t.Fatal("failed attachment remained writable")
	}
	if r.count("input/write") != 1 {
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
