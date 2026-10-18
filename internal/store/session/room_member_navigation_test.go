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

func TestRoomMemberExplicitNavigationPinnedClientFixture(t *testing.T) {
	opened := make(chan dsh.ClientFrame, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			t.Errorf("unexpected RPC %s", r.URL.Path)
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
	if s.SelectObservedRoomMember(&dsh.Client{Endpoint: client.Endpoint}, "previous", "member") || s.SelectObservedRoomMember(client, "wrong", "member") {
		t.Fatal("stale client/selection navigated")
	}
	select {
	case <-opened:
		t.Fatal("stale action followed")
	default:
	}
	if !s.SelectObservedRoomMember(client, "previous", "member") {
		t.Fatal("explicit open rejected")
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
				if json.Unmarshal(frame.Payload, &payload) != nil || payload.Args.Request.Address.Kind != "session" || payload.Args.Request.Address.SessionID != "member" {
					t.Fatal("wrong ordinary member address", string(frame.Payload))
				}
			} else if frame.Endpoint != "session/control" {
				t.Fatal("opening ran unexpected operation", frame.Endpoint)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("explicit observers did not open")
		}
	}
	if !endpoints["session/follow"] || !endpoints["session/control"] || s.Snapshot().SelectedID != "member" {
		t.Fatal("existing session observation missing")
	}
	replacement := &dsh.Client{Endpoint: client.Endpoint}
	s.Attach(replacement, nil)
	if s.SelectObservedRoomMember(client, "member", "other") {
		t.Fatal("late original client rebound")
	}
}
