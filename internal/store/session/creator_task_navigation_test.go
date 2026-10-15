package session

import (
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

// The only added fixture tests the two review boundaries, without re-running
// creation/apply or using a real Host. A notify hook forces replacement exactly
// between the atomic selection and launching the captured stream readers.
func TestCreatorTaskEntryNavigationFenceFixture(t *testing.T) {
	opened := make(chan dsh.ClientFrame, 8)
	oldHost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			t.Errorf("unexpected HTTP operation %s", r.URL.Path)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			var frame dsh.ClientFrame
			if wsjson.Read(r.Context(), conn, &frame) != nil {
				return
			}
			if frame.Type == dsh.FrameOpen {
				opened <- frame
			}
		}
	}))
	defer oldHost.Close()
	var replacementCalls atomic.Int32
	replacementHost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		replacementCalls.Add(1)
		http.Error(w, "replacement must not be contacted", 500)
	}))
	defer replacementHost.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: oldHost.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	s := New()
	s.Attach(client, nil)
	s.AttachMux(mux)
	s.SetWorkspaceCwd("/fixture")
	s.mu.Lock()
	s.selectedID = "previous"
	s.mu.Unlock()
	c := NewCreatorTask(s, nil)
	c.ObserveVisible(true)
	c.Prepare()
	c.mu.Lock()
	c.state.Review = false
	c.state.Attempted = true
	c.state.Created = true
	c.state.Applied = true
	c.mu.Unlock()
	reviewTicket := c.Snapshot().Ticket
	c.ObserveVisible(false)
	c.ObserveVisible(true) // Hide -> Show must not revive old callbacks.
	if c.Open(reviewTicket, true) {
		t.Fatal("hidden view's callback survived Show")
	}
	state := c.Snapshot()
	if !c.Open(state.Ticket, false) {
		t.Fatal("explicit retained-target open rejected")
	}
	endpoints := map[string]bool{}
	for range 2 {
		select {
		case frame := <-opened:
			endpoints[frame.Endpoint] = true
			if frame.Endpoint == "session/follow" {
				var payload struct {
					Args struct {
						Request struct {
							Address struct{ Kind, SessionID string } `json:"address"`
						} `json:"request"`
					} `json:"args"`
				}
				if json.Unmarshal(frame.Payload, &payload) != nil || payload.Args.Request.Address.SessionID != state.SessionID {
					t.Fatal("wrong captured target", string(frame.Payload))
				}
			}
		case <-time.After(3 * time.Second):
			t.Fatal("missing captured readers")
		}
	}
	if !endpoints["session/follow"] || !endpoints["session/control"] {
		t.Fatal("follow/control not both bound", endpoints)
	}
	// Start another retained-result opening, then replace both client and mux in
	// the publication callback before Open can start its observer goroutines.
	c.Prepare()
	c.mu.Lock()
	c.state.Review = false
	c.state.Attempted = true
	c.state.Created = true
	c.state.Applied = true
	c.mu.Unlock()
	next := c.Snapshot()
	replacement := &dsh.Client{Endpoint: dsh.Endpoint{Origin: replacementHost.URL}}
	replacementMux := &dsh.Mux{Endpoint: replacement.Endpoint}
	defer replacementMux.Close()
	var switched atomic.Bool
	s.mu.Lock()
	s.notify = func() {
		if switched.CompareAndSwap(false, true) {
			s.Attach(replacement, nil)
			s.AttachMux(replacementMux)
			s.mu.Lock()
			s.selectSessionLocked("replacement-session")
			s.mu.Unlock()
		}
	}
	s.mu.Unlock()
	if !c.Open(next.Ticket, false) {
		t.Fatal("second explicit admission rejected")
	}
	// The captured control reader may open on the OLD mux; the replacement is
	// never dynamically acquired. Its frame fence rejects the changed owner.
	select {
	case frame := <-opened:
		if frame.Endpoint != "session/control" {
			t.Fatal("unexpected stale reader", frame.Endpoint)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("captured old reader was not launched")
	}
	if replacementCalls.Load() != 0 || s.Snapshot().SelectedID != "replacement-session" {
		t.Fatal("observer rebound to replacement Host/selection")
	}
	s.Detach()
}
