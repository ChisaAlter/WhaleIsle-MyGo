package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"whaleisle/internal/dsh"
)

type turnDeliveryTransport func(*http.Request) (*http.Response, error)

func (f turnDeliveryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// One in-memory fixture. No real Host, listener, model, files or network.
func TestNativeTurnDeliveryChainFixture(t *testing.T) {
	event := func(raw string) Message {
		m, ok := classifyPersistenceEvent(json.RawMessage(raw))
		if !ok {
			t.Fatalf("unclassified fixture: %s", raw)
		}
		return m
	}
	messages := []Message{
		event(`{"type":"deliverables/presented","seq":8,"data":{"turn":1,"callId":"p","files":[{"path":"a.md","description":"notes; do not execute"},{"path":42},{"path":"b.md"}]}}`),
		event(`{"type":"workspace/changes","seq":9,"data":{"turn":1}}`),
		event(`{"type":"deliverables/presented","seq":10,"data":{"turn":2,"callId":"q","files":[{"path":"other.md"}]}}`),
		event(`{"type":"assistant/message","seq":11,"data":{"turn":1,"message":{"id":"reply","content":[{"type":"text","text":"done"}]}}}`),
		event(`{"type":"turn/end","seq":12,"data":{"turn":1}}`),
	}
	if got := messages[0].Files[0]; got != "a.md" {
		t.Fatalf("description polluted path: %q", got)
	}
	cards := ProjectTurnDeliveryMessages(messages)
	var delivery *TurnDelivery
	for _, m := range cards {
		if m.TurnDelivery != nil {
			if delivery != nil {
				t.Fatal("duplicate turn card")
			}
			delivery = m.TurnDelivery
		}
	}
	if delivery == nil || delivery.Turn != 1 || len(delivery.Files) != 2 || delivery.Files[1].Target.Index != 2 || delivery.Files[0].Description != "notes; do not execute" || delivery.Changes.Seq != 9 {
		t.Fatalf("projection: %#v", delivery)
	}
	if got := ProjectTurnDeliveryMessages(messages[:4]); len(got) != 1 {
		t.Fatalf("live turn gained delivery authority: %d", len(got))
	}
	late := append([]Message(nil), messages...)
	late[len(late)-1] = event(`{"type":"turn/end","seq":14,"data":{"turn":1}}`)
	late = append(late, event(`{"type":"deliverables/presented","seq":13,"data":{"turn":1,"callId":"late","files":[{"path":"late.md"}]}}`))
	if len(TurnDeliveries(late)[1].Files) != 2 {
		t.Fatal("post-closing declaration acquired an action")
	}
	s := New()
	var mu sync.Mutex
	calls := []string{}
	mode := "ok"
	started, release := make(chan struct{}), make(chan struct{})
	transport := turnDeliveryTransport(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		current := mode
		mu.Unlock()
		if r.Header.Get("Cookie") != "secret=fake" || r.URL.Host != "fixture.invalid" {
			t.Error("wrong captured Host/auth")
		}
		if q := r.URL.Query(); q.Get("sessionId") != "fork-view" && r.URL.Path != "/api/present.host" {
			t.Errorf("not viewed fork: %s", r.URL)
		}
		if current == "block" {
			close(started)
			<-release
		}
		status, body := 200, `{"name":"fixture desktop","available":true,"fileManager":"explorer"}`
		switch r.URL.Path {
		case "/api/changes.summary":
			body = `{"turn":1,"files":[{"path":"a.md","display":"a.md","added":1,"deleted":0}],"total":1,"added":1,"deleted":0}`
		case "/api/changes.diff":
			body = `{"kind":"text","path":"a.md","display":"a.md","before":true,"after":true,"coarse":false,"hunks":[{"oldStart":1,"oldLines":0,"newStart":1,"newLines":1,"lines":["+hello"]}]}`
		case "/api/present.host":
			if r.Method != http.MethodGet || r.URL.RawQuery != "" {
				t.Error("Host metadata coordinates")
			}
		case "/api/changes.open":
			status, body = 204, ""
			if r.Method != http.MethodPost || r.URL.Query().Get("seq") != "9" || r.URL.Query().Get("index") != "0" {
				t.Error("changed-file native coordinates")
			}
		case "/api/present.open":
			status, body = 204, ""
			if r.URL.Query().Get("seq") != "8" || r.URL.Query().Get("index") != "2" || r.URL.Query().Get("action") != "reveal" {
				t.Errorf("wrong file coordinates: %s", r.URL)
			}
		default:
			t.Fatalf("invented route: %s", r.URL)
		}
		if current == "malformed" {
			body = `{"turn":1,"files":[],"total":0}`
		}
		if current == "fail" {
			status = 500
		}
		if current == "missing" {
			status = 404
		}
		if current == "mismatch" {
			body = `{"turn":2,"files":[],"total":0,"added":0,"deleted":0}`
		}
		header := http.Header{}
		if current == "redirect" {
			status = 302
			header.Set("Location", "https://other.invalid/steal")
		}
		return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://fixture.invalid", Cookie: "secret=fake"}, HTTP: &http.Client{Transport: transport}}
	s.client = client
	s.selectedID = "fork-view"
	s.followGeneration = 3
	s.planHistoryBinding = fmt.Sprintf("%p\x00%s", client, s.selectedID)
	s.turnFilesHistoryBinding = turnFileHistoryBinding(client, s.selectedID)
	s.messages = messages
	scope := s.Snapshot().TurnFiles.Scope
	count := func() int { mu.Lock(); defer mu.Unlock(); return len(calls) }
	setMode := func(v string) { mu.Lock(); mode = v; mu.Unlock() }
	if count() != 0 {
		t.Fatal("snapshot initiated reads")
	}
	summary := TurnFileRequest{Target: *delivery.Changes, Action: "summary"}
	s.RunTurnFileAction(context.Background(), scope, summary)
	state := s.Snapshot().TurnFiles.Results[summary]
	if !state.Ready || state.Summary == nil || state.Summary.Turn != 1 {
		t.Fatalf("summary: %#v", state)
	}
	host := TurnFileRequest{Target: *delivery.Changes, Action: "host"}
	s.RunTurnFileAction(context.Background(), scope, host)
	if got := s.Snapshot().TurnFiles.Results[host]; got.Host == nil || !got.Host.Available {
		t.Fatal("Host metadata missing")
	}
	diff := TurnFileRequest{Target: *delivery.Changes, Action: "diff"}
	diff.Target.Index = 0
	s.RunTurnFileAction(context.Background(), scope, diff)
	if got := s.Snapshot().TurnFiles.Results[diff]; !got.Ready || got.Diff == nil || !strings.Contains(got.Diff.Patch(), "+hello") {
		t.Fatalf("diff: %#v", got)
	}
	changedOpen := diff
	changedOpen.Action = "open"
	s.RunTurnFileAction(context.Background(), scope, changedOpen)
	if !s.Snapshot().TurnFiles.Results[changedOpen.StateKey()].Ready {
		t.Fatal("changed-file native action failed")
	}
	invalid := diff
	invalid.Target.Index = 2
	before := count()
	s.RunTurnFileAction(context.Background(), scope, invalid)
	if count() != before {
		t.Fatal("unlisted index read")
	}
	path, ok := s.TurnFilePreviewPath(scope, delivery.Files[0].Target)
	if !ok || path != "a.md" {
		t.Fatal("preview did not preserve path")
	}
	setMode("fail")
	open := TurnFileRequest{Target: delivery.Files[1].Target, Action: "reveal"}
	s.RunTurnFileAction(context.Background(), scope, open)
	key := open.StateKey()
	if got := s.Snapshot().TurnFiles.Results[key]; got.Error == "" || got.Action != "reveal" {
		t.Fatalf("file failure: %#v", got)
	}
	before = count()
	_ = s.Snapshot()
	if count() != before {
		t.Fatal("automatic retry")
	}
	setMode("ok")
	s.RunTurnFileAction(context.Background(), scope, open)
	if got := s.Snapshot().TurnFiles.Results[key]; !got.Ready || got.Error != "" {
		t.Fatalf("explicit retry: %#v", got)
	}
	setMode("redirect")
	s.RunTurnFileAction(context.Background(), scope, summary)
	if got := s.Snapshot().TurnFiles.Results[summary]; got.Error == "" {
		t.Fatal("redirect followed")
	}
	setMode("malformed")
	s.RunTurnFileAction(context.Background(), scope, summary)
	if got := s.Snapshot().TurnFiles.Results[summary]; got.Error == "" || got.Summary != nil {
		t.Fatal("malformed summary accepted")
	}
	setMode("mismatch")
	s.RunTurnFileAction(context.Background(), scope, summary)
	if got := s.Snapshot().TurnFiles.Results[summary]; got.Summary != nil || got.Error == "" {
		t.Fatal("foreign turn adopted")
	}
	setMode("missing")
	s.RunTurnFileAction(context.Background(), scope, summary)
	if got := s.Snapshot().TurnFiles.Results[summary]; !got.Unavailable || got.Summary != nil {
		t.Fatal("stale summary retained")
	}
	setMode("block")
	done := make(chan struct{})
	go func() { s.RunTurnFileAction(context.Background(), scope, open); close(done) }()
	<-started
	before = count()
	s.RunTurnFileAction(context.Background(), scope, open)
	other := open
	other.Action = "open"
	s.RunTurnFileAction(context.Background(), scope, other)
	if count() != before {
		t.Fatal("duplicate native request")
	}
	s.mu.Lock()
	s.invalidateFollowLocked()
	s.client = &dsh.Client{Endpoint: client.Endpoint, HTTP: client.HTTP}
	s.mu.Unlock()
	close(release)
	<-done
	if got := s.Snapshot().TurnFiles; len(got.Results) != 0 || got.Scope.Binding != "" {
		t.Fatal("old Host response escaped")
	}
	before = count()
	s.RunTurnFileAction(context.Background(), scope, open)
	if _, ok := s.TurnFilePreviewPath(scope, delivery.Files[0].Target); ok || count() != before {
		t.Fatal("stale callback actionable")
	}
	s.mu.Lock()
	s.client = client
	s.planHistoryBinding = fmt.Sprintf("%p\x00%s", client, s.selectedID)
	s.turnFilesHistoryBinding = turnFileHistoryBinding(client, s.selectedID)
	s.mu.Unlock()
	next := s.Snapshot().TurnFiles.Scope
	before = count()
	s.RunTurnFileAction(context.Background(), scope, summary)
	if next == scope || count() != before {
		t.Fatal("old generation accepted")
	}
	client.Endpoint.Cookie = "replacement=fake"
	if got := s.Snapshot().TurnFiles; got.Scope.Binding != "" {
		t.Fatal("auth replacement borrowed old history")
	}
	s.RunTurnFileAction(context.Background(), next, summary)
	if count() != before {
		t.Fatal("old auth callback sent")
	}
	client.Endpoint.Cookie = "secret=fake"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.RunTurnFileAction(ctx, next, summary)
	if count() != before {
		t.Fatal("cancelled click issued request")
	}
}
