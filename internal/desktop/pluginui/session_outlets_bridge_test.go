package pluginui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestSessionTurnTailSurfaceWireAndRetirement(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "s", Epoch: "3:4"}
	row := SessionOutlet{Handle: "tail", Slot: "conversation.chat.turnTail", Turn: 0, HasTurn: true, TurnTailSeq: 19, HasTurnTail: true, Available: true}
	state := Snapshot{Revision: 1, Subject: &subject, Surface: row.Slot, Outlets: []SessionOutlet{row}, SessionSurfaceHandle: row.Handle, SessionSurfaceRequest: "nonce"}
	if !validSnapshot(state) {
		t.Fatal("current surface refused")
	}
	for _, mode := range []string{"missing", "wrong-slot", "unavailable", "no-subject", "request-only"} {
		st := state
		st.Outlets = append([]SessionOutlet(nil), state.Outlets...)
		switch mode {
		case "missing":
			st.SessionSurfaceHandle = "missing"
		case "wrong-slot":
			st.Surface = "conversation.session.header.actions"
		case "unavailable":
			st.Outlets[0].Available = false
		case "no-subject":
			st.Subject = nil
		case "request-only":
			st.SessionSurfaceHandle = ""
		}
		if validSnapshot(st) {
			t.Fatal("dangling surface accepted", mode)
		}
	}
	o := &Owner{selected: subject, state: state}
	o.incompatible = true
	if got := o.Snapshot(); got.SessionSurfaceHandle != "" || got.SessionSurfaceRequest != "" {
		t.Fatal("incompatible snapshot kept ack")
	}
	o.incompatible = false
	o.Select(Subject{Kind: "session", SessionID: "s", Epoch: "3:5"})
	if got := o.Snapshot(); got.SessionSurfaceHandle != "" || got.SessionSurfaceRequest != "" {
		t.Fatal("new selection kept ack")
	}
	o.state = state
	o.acceptSessionResponse(&http.Response{Header: http.Header{"X-Whale-Session-Incompatible": []string{"1"}}})
	if o.state.SessionSurfaceHandle != "" || o.state.SessionSurfaceRequest != "" {
		t.Fatal("incompatible response kept ack")
	}
	o.Close()
	if got := o.Snapshot(); got.SessionSurfaceHandle != "" || got.SessionSurfaceRequest != "" {
		t.Fatal("closed owner kept ack")
	}
}

func TestSessionSameLabelOutletsDispatchOnlyCurrentOccurrence(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "s", Epoch: "3:4"}
	o := &Owner{selected: subject}
	defer o.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !o.serve(w, r) {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connect := func() *websocket.Conn {
		t.Helper()
		conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+bridgePath, &websocket.DialOptions{Subprotocols: []string{bridgeProtocol}, HTTPHeader: http.Header{"Origin": []string{server.URL}}})
		if err != nil {
			t.Fatal(err)
		}
		var selection map[string]any
		if err := wsjson.Read(ctx, conn, &selection); err != nil || selection["type"] != "select" {
			t.Fatalf("initial selection %v: %v", selection, err)
		}
		return conn
	}
	conn := connect()
	defer conn.CloseNow()
	rows := []SessionOutlet{
		{Handle: "turn0", Slot: "conversation.chat.turnTail", Label: "Deliverable", Turn: 0, HasTurn: true, TurnTailSeq: 19, HasTurnTail: true, Available: true},
		{Handle: "turn2", Slot: "conversation.chat.turnTail", Label: "Deliverable", Turn: 2, HasTurn: true, Available: true},
		{Handle: "call-a", Slot: "tool.call.toolview", Label: "Tool output", CallID: "call-a", Available: true},
		{Handle: "call-b", Slot: "tool.call.toolview", Label: "Tool output", CallID: "call-b", Available: true},
		{Handle: "disabled", Slot: "tool.call.toolview", Label: "Tool output", CallID: "unavailable", Available: false},
	}
	publish := func(c *websocket.Conn, revision uint64, outlets []SessionOutlet) Snapshot {
		t.Helper()
		message := struct {
			Type string `json:"type"`
			Snapshot
		}{Type: "state", Snapshot: Snapshot{Revision: revision, Subject: &subject, Outlets: outlets}}
		if len(outlets) > 0 && outlets[0].Available {
			message.Surface, message.SessionSurfaceHandle, message.SessionSurfaceRequest = outlets[0].Slot, outlets[0].Handle, "fixture-ack"
		}
		if err := wsjson.Write(ctx, c, message); err != nil {
			t.Fatal(err)
		}
		for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
			if s := o.Snapshot(); s.Revision == revision && s.Connected {
				return s
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("current outlet state did not settle")
		return Snapshot{}
	}
	expect := func(c *websocket.Conn, s Snapshot, handle string) {
		t.Helper()
		if err := o.ShowSessionOutlet(s.Page, s.Revision, subject, handle); err != nil {
			t.Fatal(err)
		}
		var command struct {
			Type     string
			Revision uint64
			Subject  Subject
			Handle   string
		}
		if err := wsjson.Read(ctx, c, &command); err != nil {
			t.Fatal(err)
		}
		if command.Type != "sessionSurface" || command.Revision != s.Revision || command.Subject != subject || command.Handle != handle {
			t.Fatalf("wrong occurrence dispatched: %+v", command)
		}
	}
	first := publish(conn, 1, rows)
	// The canonical request uses the same live page and is echoed verbatim.
	if err := o.ShowSessionTurnTail(first.Page, first.Revision, subject, "turn0", SessionTurnTail{Turn: 0, Seq: 19}, "request-1"); err != nil {
		t.Fatal(err)
	}
	var tailCommand struct {
		Type     string
		Request  string
		TurnTail SessionTurnTail
	}
	if err := wsjson.Read(ctx, conn, &tailCommand); err != nil || tailCommand.Type != "sessionSurface" || tailCommand.Request != "request-1" || tailCommand.TurnTail.Seq != 19 {
		t.Fatalf("canonical command %+v %v", tailCommand, err)
	}
	if o.ShowSessionTurnTail(first.Page, first.Revision, subject, "turn0", SessionTurnTail{Turn: 0, Seq: 18}, "request-2") == nil {
		t.Fatal("wrong canonical end seq accepted")
	}
	if o.ShowSessionTurnTail(first.Page, first.Revision, subject, "turn0", SessionTurnTail{Turn: 0, Seq: 19}, "") == nil {
		t.Fatal("unacknowledged request accepted")
	}
	originalPage := o.page
	o.mu.Lock()
	o.page = &bridgePage{socket: originalPage.socket, id: first.Page + 1}
	o.mu.Unlock()
	if o.ShowSessionOutlet(first.Page, first.Revision, subject, "turn0") == nil {
		t.Fatal("old snapshot redirected to replacement socket")
	}
	o.mu.Lock()
	o.page = originalPage
	o.state.Connected = false
	o.mu.Unlock()
	if o.ShowSessionOutlet(first.Page, first.Revision, subject, "turn0") == nil {
		t.Fatal("disconnected owner accepted outlet")
	}
	o.mu.Lock()
	o.state.Connected = true
	o.mu.Unlock()
	for _, handle := range []string{"turn0", "turn2", "call-a", "call-b"} {
		expect(conn, first, handle)
	}
	for _, handle := range []string{"disabled", "missing"} {
		if o.ShowSessionOutlet(first.Page, first.Revision, subject, handle) == nil {
			t.Fatalf("%s occurrence accepted", handle)
		}
	}
	other := subject
	other.Epoch = "3:5"
	if o.ShowSessionOutlet(first.Page, first.Revision, other, "turn0") == nil {
		t.Fatal("old selection acquired another occurrence")
	}
	rows[0], rows[3] = rows[3], rows[0]
	second := publish(conn, 2, rows)
	if o.ShowSessionOutlet(first.Page, first.Revision, subject, "turn0") == nil {
		t.Fatal("obsolete revision accepted")
	}
	expect(conn, second, "turn0")
	third := publish(conn, 3, []SessionOutlet{rows[0]})
	if o.ShowSessionOutlet(third.Page, third.Revision, subject, "turn0") == nil {
		t.Fatal("withdrawn same-label occurrence accepted")
	}
	expect(conn, third, "call-b")
	newPage := connect()
	defer newPage.CloseNow()
	if got := o.Snapshot(); got.SessionSurfaceHandle != "" || got.SessionSurfaceRequest != "" {
		t.Fatal("replacement page retained prior surface acknowledgement")
	}
	current := publish(newPage, 1, []SessionOutlet{rows[0]})
	if current.Page == third.Page {
		t.Fatal("replacement page retained identity")
	}
	if o.ShowSessionOutlet(third.Page, third.Revision, subject, "call-b") == nil {
		t.Fatal("prior page accepted")
	}
	expect(newPage, current, "call-b")
	foreign := subject
	foreign.Epoch = "3:5"
	if err := wsjson.Write(ctx, newPage, struct {
		Type string `json:"type"`
		Snapshot
	}{Type: "state", Snapshot: Snapshot{Revision: 2, Subject: &foreign, Surface: rows[0].Slot, Outlets: []SessionOutlet{rows[0]}, SessionSurfaceHandle: rows[0].Handle, SessionSurfaceRequest: "foreign"}}); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); ; {
		got := o.Snapshot()
		if got.Revision == 2 {
			if got.Subject != nil || got.SessionSurfaceHandle != "" || got.SessionSurfaceRequest != "" {
				t.Fatal("foreign subject retained surface acknowledgement")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("foreign snapshot did not settle")
		}
		time.Sleep(time.Millisecond)
	}
	o.Close()
	if o.ShowSessionOutlet(current.Page, current.Revision, subject, "call-b") == nil {
		t.Fatal("closed owner retained occurrence")
	}
}
