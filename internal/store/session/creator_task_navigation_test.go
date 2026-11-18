package session

import (
	"context"
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
	// Those captured readers open the old Host best-effort: runControlWatch
	// returns silently when the socket is already gone, so this asserts the
	// binding they cannot escape, never that a frame arrives.
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
	// The captured readers were bound under the old owner, so they can never
	// reach the replacement Host or the replacement selection. Whether they
	// still open the old mux afterwards depends on that socket, which the store
	// does not own; TestControlReaderFramesAreFencedAcrossOwnerSwap covers the
	// receive fence on a reader that provably opened.
	if replacementCalls.Load() != 0 || s.Snapshot().SelectedID != "replacement-session" {
		t.Fatal("observer rebound to replacement Host/selection")
	}
	s.Detach()
}

// A captured control reader must fold nothing once another owner holds the
// store. Swap the owner while the reader is provably open on a live Host, then
// let the Host push a frame: the reader's fence rejects it and the reader
// exits, which the Host observes as the stream's cancel.
func TestControlReaderFramesAreFencedAcrossOwnerSwap(t *testing.T) {
	controlOpen := make(chan struct{})
	sendFrame := make(chan struct{})
	readerExited := make(chan struct{})
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		var open dsh.ClientFrame
		if wsjson.Read(r.Context(), conn, &open) != nil || open.Endpoint != "session/control" {
			return
		}
		close(controlOpen)
		<-sendFrame
		if wsjson.Write(r.Context(), conn, dsh.ServerFrame{
			Type:     dsh.FrameItem,
			StreamID: open.StreamID,
			Value:    json.RawMessage(`{"type":"session-presentation","note":"stale-owner"}`),
		}) != nil {
			return
		}
		for {
			var frame dsh.ClientFrame
			if wsjson.Read(r.Context(), conn, &frame) != nil {
				return
			}
			if frame.Type == dsh.FrameCancel && frame.StreamID == open.StreamID {
				close(readerExited)
				return
			}
		}
	}))
	defer host.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: host.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	s := New()
	s.Attach(client, nil)
	s.AttachMux(mux)
	s.mu.Lock()
	s.selectedID = "A"
	s.mu.Unlock()
	s.WatchControl(context.Background())
	wait(t, controlOpen)

	replacement := &dsh.Client{Endpoint: dsh.Endpoint{Origin: host.URL}}
	s.Attach(replacement, nil)
	s.mu.Lock()
	s.selectSessionLocked("B")
	s.mu.Unlock()
	close(sendFrame)
	wait(t, readerExited)
	if note := s.Snapshot().ControlNote; note != "" {
		t.Fatalf("stale control frame folded after the owner swap: %q", note)
	}
	s.Detach()
}
