package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func publishedFixtureStore() *Store {
	s := New()
	s.mu.Lock()
	s.client = &dsh.Client{}
	s.describedClient = s.client
	s.hasDocument = true
	s.namespaces = []Namespace{
		{NS: "general", Revision: 1, Fields: []Field{{Key: "name", Kind: FieldString}}, Value: json.RawMessage(`{"name":"saved"}`)},
		{NS: "ui-chat", Revision: 3, Value: json.RawMessage(`{}`)},
	}
	s.selectedNS = "general"
	s.unlockPublish()
	return s
}

func TestSettingsSnapshotSharedUntilWrite(t *testing.T) {
	s := publishedFixtureStore()
	first := s.Snapshot()
	if first != s.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
	// Read-only sections compare and publish nothing.
	s.NamespaceValue("general")
	s.SelectedNamespace()
	s.EnsureLoaded(context.Background())
	if s.Snapshot() != first {
		t.Fatal("a section that changed nothing published a new version")
	}
	if got := testing.AllocsPerRun(100, func() { s.SelectedNamespace() }); got != 0 {
		t.Fatalf("a read-only section allocates %g per run", got)
	}
}

func TestSettingsWritePublishesAndSignals(t *testing.T) {
	s := publishedFixtureStore()
	before := s.Snapshot()
	changed := s.Changed()
	s.SetDraft("name", "draft")
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 || after.Value.Drafts["name"] != "draft" || !after.Value.Dirty {
		t.Fatalf("SetDraft did not publish Rev+1: %d -> %d %+v", before.Rev, after.Rev, after.Value.Drafts)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
	if before.Value.Drafts["name"] != "" {
		t.Fatal("a draft edit reached an earlier version")
	}
}

// Writers edit namespace elements through findNamespaceLocked pointers and
// replace their payloads; a version a reader holds must keep its values.
func TestSettingsNamespaceEditLeavesHeldVersion(t *testing.T) {
	s := publishedFixtureStore()
	held := s.Snapshot()
	s.mu.Lock()
	ns := s.findNamespaceLocked("general")
	ns.Revision = 2
	ns.Value = json.RawMessage(`{"name":"next"}`)
	s.unlockPublish()
	if held.Value.Namespaces[0].Revision != 1 || string(held.Value.Namespaces[0].Value) != `{"name":"saved"}` {
		t.Fatalf("namespace edit reached a published version: %+v", held.Value.Namespaces[0])
	}
	next := s.Snapshot()
	if next == held || next.Value.Namespaces[0].Revision != 2 || string(next.Value.Namespaces[0].Value) != `{"name":"next"}` {
		t.Fatal("namespace edit was not published")
	}
}

func TestSettingsStaleDescribeDroppedAndSnapshotUnblocked(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	first := true
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://published.invalid"}, HTTP: &http.Client{Transport: viewTabsDescriptionTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		_ = json.NewDecoder(r.Body).Decode(&request)
		if first {
			first = false
			close(started)
			<-release
		}
		recorder := httptest.NewRecorder()
		value := map[string]any{"writable": true, "namespaces": []any{}}
		_ = json.NewEncoder(recorder).Encode(map[string]any{"type": "server-response", "rpcId": request["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return recorder.Result(), nil
	})}}
	s := New()
	s.Attach(client, nil)
	s.RefreshDescribe(context.Background())
	<-started
	read := make(chan bool, 1)
	go func() { read <- s.Snapshot().Value.Loading }()
	select {
	case loading := <-read:
		if !loading {
			t.Fatal("in-flight describe not published as Loading")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Snapshot blocked behind settings/describe")
	}
	s.Detach() // retires the describe generation
	held := s.Snapshot()
	close(release)
	time.Sleep(50 * time.Millisecond)
	if got := s.Snapshot(); got != held || got.Value.Described {
		t.Fatal("stale describe completion was published")
	}
}
