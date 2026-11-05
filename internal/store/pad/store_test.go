package pad

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func samplePanel(id string) Panel {
	return Panel{ID: id, Version: 1, Definition: Definition{Title: id, HTML: "<b id=out>panel</b>", Script: "dsh.onData(function(){})", Height: json.RawMessage(`"auto"`), Sources: []Source{}}, Data: Data{Sources: map[string]Reading{}}}
}
func waitState(t *testing.T, s *Store, changed <-chan struct{}, predicate func(Snapshot) bool) Snapshot {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		snap := s.Snapshot()
		if predicate(snap) {
			return snap
		}
		select {
		case <-changed:
		case <-timer.C:
			t.Fatalf("Pad state not reached: %+v", snap)
		}
	}
}
func TestPadStoreLiveWatchActionsAndGeneration(t *testing.T) {
	changed := make(chan struct{}, 64)
	watches := make(chan string, 16)
	actions := make(chan Action, 8)
	p := samplePanel("one")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "dsh-auth=qa" {
			t.Error("session cookie missing")
		}
		switch r.URL.Path {
		case "/api/dsh-pad.stream":
			w.Header().Set("Content-Type", "text/event-stream")
			watches <- r.URL.Query().Get("watch")
			raw, _ := json.Marshal(streamEvent{Type: "panels", Panels: []Panel{p}})
			fmt.Fprintf(w, "data: %s\n\n", raw)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/api/dsh-pad.action":
			var a Action
			_ = json.NewDecoder(r.Body).Decode(&a)
			actions <- a
			fmt.Fprint(w, `{"ok":true}`)
		case "/api/dsh-pad.models":
			fmt.Fprint(w, `{"default":{"provider":"qa","model":"tiny"},"providers":[]}`)
		case "/api/dsh-pad.run":
			fmt.Fprint(w, `{"ok":true,"text":"local-result"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	s, err := Open(t.TempDir(), func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Detach()
	client, _ := NewClient(server.URL, "dsh-auth=qa")
	s.Attach(client, 7)
	waitState(t, s, changed, func(v Snapshot) bool { return v.Loaded })
	if watch := <-watches; watch != "" {
		t.Fatal(watch)
	}
	s.UpdateLayout(func(l *Layout) { l.Expanded = true })
	select {
	case watch := <-watches:
		if watch != "one" {
			t.Fatal(watch)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expanded dock did not watch panel")
	}
	s.Do(Action{Generation: 7, Op: "approve", PanelID: "one", Fingerprint: "exact"})
	select {
	case a := <-actions:
		if a.Fingerprint != "exact" || a.Op != "approve" {
			t.Fatal(a)
		}
	case <-time.After(time.Second):
		t.Fatal("approval not sent")
	}
	s.LoadModels()
	waitState(t, s, changed, func(v Snapshot) bool { return v.Catalog != nil })
	result, err := s.RunForGeneration(context.Background(), 7, "one", "summarize", "system")
	if err != nil || result.Text != "local-result" {
		t.Fatal(result, err)
	}
	s.mu.Lock()
	oldEpoch, oldStream := s.epoch, s.streamID
	s.mu.Unlock()
	s.Attach(client, 8)
	if _, err := s.RunForGeneration(context.Background(), 7, "one", "old", ""); err == nil {
		t.Fatal("old frame used new Host")
	}
	s.receive(oldEpoch, oldStream, streamEvent{Type: "panels", Panels: []Panel{samplePanel("stale")}})
	if _, ok := s.Panel("stale"); ok {
		t.Fatal("old stream leaked into new generation")
	}
	if err := s.SetPanelStateForGeneration(7, "one", "tab", json.RawMessage(`"old"`)); err == nil {
		t.Fatal("stale state persisted")
	}
}
func TestPadLayoutDropStateAndHistoryRestore(t *testing.T) {
	home := t.TempDir()
	s, err := Open(home, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.panels = []Panel{samplePanel("a"), samplePanel("b"), samplePanel("c")}
	s.layout.Tree = normalizeTree(nil, s.panels)
	s.DropCard("a", "b", "left")
	tree := s.Snapshot().Layout.Tree
	if tree.Kind != "split" || tree.Children[0].Dir != "row" {
		t.Fatalf("no nested horizontal split: %+v", tree)
	}
	s.DropCard("a", "b", "center")
	s.UpdateLayout(func(l *Layout) { l.Expanded = true; l.Width = 480; l.Heights["b"] = 240 })
	if err := s.SetPanelState("b", "tab", json.RawMessage(`"details"`)); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(home, nil)
	if err != nil {
		t.Fatal(err)
	}
	snap := restored.Snapshot()
	if !snap.Layout.Expanded || snap.Layout.Width != 480 || snap.Layout.Heights["b"] != 240 || string(restored.PanelState("b")["tab"]) != `"details"` {
		t.Fatal(snap.Layout)
	}
	if err := s.SetPanelState("b", "huge", json.RawMessage(`"`+strings.Repeat("x", 65<<10)+`"`)); err == nil {
		t.Fatal("oversize state accepted")
	}
	p := samplePanel("trend")
	for i := 0; i < 130; i++ {
		record(&p, Data{Sources: map[string]Reading{"cpu": {OK: true, Value: json.RawMessage(`42`), At: fmt.Sprint(i)}}})
	}
	record(&p, Data{Sources: map[string]Reading{"cpu": {OK: false, At: "bad"}}})
	if len(p.History["cpu"]) != 120 || p.History["cpu"][0].At != "10" {
		t.Fatal(p.History)
	}
}
func TestPadClientRejectsExternalAndKeepsHostErrors(t *testing.T) {
	for _, origin := range []string{"https://example.com", "http://127.0.0.1:8080/path", "http://user@127.0.0.1:8080", "http://127.0.0.1:8080?x=1"} {
		if _, err := NewClient(origin, "cookie"); err == nil {
			t.Fatal(origin)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		fmt.Fprint(w, `{"ok":false,"error":"数据源已经变了，请重新确认"}`)
	}))
	defer server.Close()
	c, _ := NewClient(server.URL, "")
	if err := c.Act(context.Background(), Action{Op: "approve", PanelID: "p"}); err == nil || !strings.Contains(err.Error(), "重新确认") {
		t.Fatal(err)
	}
}
func TestPadDisableEnableAndUnfoldFalse(t *testing.T) {
	var disabled atomic.Bool
	disabled.Store(true)
	changed := make(chan struct{}, 32)
	actions := make(chan map[string]json.RawMessage, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/dsh-pad.action" {
			var body map[string]json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&body)
			actions <- body
			fmt.Fprint(w, `{"ok":true}`)
			return
		}
		if disabled.Load() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"panels\",\"panels\":[]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	s, err := Open(t.TempDir(), func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Detach()
	c, _ := NewClient(server.URL, "")
	s.Attach(c, 4)
	waitState(t, s, changed, func(v Snapshot) bool { return v.Disabled })
	disabled.Store(false)
	s.Reconnect()
	waitState(t, s, changed, func(v Snapshot) bool { return v.Loaded && !v.Disabled })
	s.Do(Action{Generation: 4, Op: "collapse", PanelID: "one", Collapsed: false})
	select {
	case body := <-actions:
		if string(body["collapsed"]) != "false" {
			t.Fatalf("unfold did not send required false boolean: %s", body)
		}
	case <-time.After(time.Second):
		t.Fatal("unfold not sent")
	}
}

func TestModelSelectionWaitsForBusinessResultAndRetiresOnHostChange(t *testing.T) {
	changed := make(chan struct{}, 32)
	started := make(chan Action, 4)
	replies := make(chan string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dsh-pad.action" {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"type\":\"panels\",\"panels\":[]}\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		var a Action
		_ = json.NewDecoder(r.Body).Decode(&a)
		started <- a
		select {
		case reply := <-replies:
			fmt.Fprint(w, reply)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	s, err := Open(t.TempDir(), func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Detach()
	c, _ := NewClient(server.URL, "")
	s.Attach(c, 7)
	request := s.ChooseModel(7, "panel", &ModelSelection{Provider: "fixture", Model: "small"})
	if request == 0 || !s.Snapshot().ModelActions["panel"].Busy || s.ChooseModel(7, "panel", nil) != 0 {
		t.Fatal("selection was not serialized")
	}
	if a := <-started; a.Op != "model" || a.Model.Model != "small" {
		t.Fatal(a)
	}
	replies <- `{"ok":false,"error":"model unavailable"}`
	waitState(t, s, changed, func(v Snapshot) bool { return v.ModelActions["panel"].Error != "" })
	if a := s.Snapshot().ModelActions["panel"]; a.Busy || a.Succeeded || a.Request != request || a.Error != "model unavailable" {
		t.Fatal("business rejection counted as success", a)
	}
	request = s.ChooseModel(7, "panel", nil)
	<-started
	replies <- `{"ok":true}`
	waitState(t, s, changed, func(v Snapshot) bool { return v.ModelActions["panel"].Succeeded })
	if a := s.Snapshot().ModelActions["panel"]; a.Request != request || a.Busy || a.Error != "" {
		t.Fatal(a)
	}
	s.ChooseModel(7, "panel", nil)
	<-started
	s.Attach(c, 8)
	if len(s.Snapshot().ModelActions) != 0 || s.ChooseModel(7, "panel", nil) != 0 {
		t.Fatal("old Host selection survived replacement")
	}
}
