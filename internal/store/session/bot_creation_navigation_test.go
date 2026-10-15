package session

import (
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

func TestBotCreationNavigationCapturedIdentity(t *testing.T) {
	opened := make(chan dsh.ClientFrame, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			t.Errorf("unexpected %s", r.URL.Path)
			http.Error(w, "unexpected", 500)
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
	defer srv.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	defer mux.Close()
	s := New()
	s.Attach(client, nil)
	s.AttachMux(mux)
	s.mu.Lock()
	s.selectedID = "previous"
	s.mu.Unlock()
	target := s.CaptureBotCreationOpen(client)
	if target == nil || !target.Current() {
		t.Fatal("no capture")
	}
	s.mu.Lock()
	s.selectSessionLocked("other")
	s.selectSessionLocked("previous")
	s.mu.Unlock()
	if target.Current() || target.Select("canonical") {
		t.Fatal("selection ABA revived target")
	}
	target = s.CaptureBotCreationOpen(client)
	replacement := &dsh.Client{Endpoint: client.Endpoint}
	s.Attach(replacement, nil)
	if target.Current() || target.Select("canonical") {
		t.Fatal("Host generation rebound")
	}
	s.Attach(client, nil)
	s.AttachMux(mux)
	target = s.CaptureBotCreationOpen(client)
	if target == nil || !target.Select("canonical") {
		t.Fatal("explicit select failed")
	}
	if target.Current() || target.Select("canonical-again") {
		t.Fatal("replayed navigation")
	}
	endpoints := map[string]bool{}
	for range 2 {
		select {
		case frame := <-opened:
			endpoints[frame.Endpoint] = true
			var p struct {
				Args struct {
					Request struct {
						Address struct{ Kind, SessionID string } `json:"address"`
					} `json:"request"`
				} `json:"args"`
			}
			if frame.Endpoint == "session/follow" {
				if json.Unmarshal(frame.Payload, &p) != nil || p.Args.Request.Address.Kind != "session" || p.Args.Request.Address.SessionID != "canonical" {
					t.Fatal("wrong follow", string(frame.Payload))
				}
			} else if frame.Endpoint != "session/control" {
				t.Fatal("unexpected action", frame.Endpoint)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("missing observer")
		}
	}
	if !endpoints["session/follow"] || !endpoints["session/control"] || s.Snapshot().SelectedID != "canonical" {
		t.Fatal("observers not pinned")
	}
	s.Attach(nil, nil)
}
