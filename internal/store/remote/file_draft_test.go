package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

type capturedDraftFixture struct {
	mu   sync.Mutex
	text string
}

func (b *capturedDraftFixture) CommittedText() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.text
}
func (b *capturedDraftFixture) ReplaceText(text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.text = text
}

func seededDraftFiles(t *testing.T, origin string) (*Files, *capturedDraftFixture) {
	t.Helper()
	s := NewFiles()
	s.Bind(&dsh.Client{Endpoint: dsh.Endpoint{Origin: origin}}, "session-a", nil)
	mtime := float64(10)
	v := s.states[s.key]
	v.Bound, v.Loaded, v.FileLoaded = true, true, true
	v.FilePath, v.Draft, v.RemoteIdentity = "/work/a.txt", "base", "remote-a"
	v.File = RemoteFile{Content: "base", Mtime: &mtime}
	stashResource(v)
	b := &capturedDraftFixture{text: "base"}
	if got := s.EditBuffer(s.Snapshot(), func(string) FileDraftBuffer { return b }); got != b {
		t.Fatal("could not attach captured resource")
	}
	return s, b
}

func TestCapturedRemoteDraftParticipatesInCloseWithoutWidgetCallback(t *testing.T) {
	s, b := seededDraftFiles(t, "http://unused.invalid")
	b.ReplaceText("native committed before frame")
	if !s.HasDirty() {
		t.Fatal("native commit was invisible to pane dirty guard")
	}
	if err := s.PreflightDirtyForClose(); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	plan, err := s.preflightCloseSaveLocked()
	s.mu.Unlock()
	if err != nil || plan == nil || len(plan.resources) != 1 || plan.resources[0].Draft != "native committed before frame" {
		t.Fatalf("close captured stale text: %#v %v", plan, err)
	}
	if done, err := s.CheckDirtyForClose(); done || err == nil {
		t.Fatal("unsaved native text allowed close", done, err)
	}
	s.RequestClose("/work/a.txt")
	if s.Snapshot().PendingClose != "/work/a.txt" {
		t.Fatal("native-only dirty tab bypassed review")
	}
	s.ConfirmClose(false)
	if s.Snapshot().Draft != "native committed before frame" {
		t.Fatal("cancel close lost committed text")
	}
}

func TestCapturedRemoteDraftStaysWithTabAndSessionOnSuspend(t *testing.T) {
	s, first := seededDraftFiles(t, "http://unused.invalid")
	v := s.states[s.key]
	mtime := float64(20)
	v.Tabs = append(v.Tabs, FileResource{Path: "/work/b.txt", Draft: "second", Loaded: true, File: RemoteFile{Content: "second", Mtime: &mtime}})
	first.ReplaceText("first tab native")
	s.Open("/work/b.txt")
	second := &capturedDraftFixture{text: "second"}
	if s.EditBuffer(s.Snapshot(), func(string) FileDraftBuffer { return second }) != second {
		t.Fatal("second tab source missing")
	}
	second.ReplaceText("second tab native")
	s.Open("/work/a.txt")
	if s.Snapshot().Draft != "first tab native" {
		t.Fatal("tab switch adopted another resource's text")
	}
	s.Suspend()
	if !s.HasDirty() {
		t.Fatal("suspend lost cached committed drafts")
	}
	s.Bind(s.client, "session-b", nil)
	first.ReplaceText("late old session input")
	if snap := s.Snapshot(); snap.SessionID != "session-b" || snap.Draft != "" {
		t.Fatal("old source wrote into new session", snap)
	}
	if err := s.PreflightDirtyForClose(); err == nil || !strings.Contains(err.Error(), "session-a") {
		t.Fatal("foreign owner borrowed active session", err)
	}
	s.Bind(s.client, "session-a", nil)
	if s.Snapshot().Draft != "late old session input" {
		t.Fatal("cached owner lost its own native text")
	}
	s.Open("/work/b.txt")
	if s.Snapshot().Draft != "second tab native" {
		t.Fatal("other cached resource was overwritten")
	}
}

func TestRemoteBufferAttachRejectsReopenedResourceAndChangedIdentity(t *testing.T) {
	s, b := seededDraftFiles(t, "http://unused.invalid")
	captured := s.Snapshot()
	s.RequestClose(captured.FilePath)
	v := s.states[s.key]
	v.FilePath, v.Draft, v.FileLoaded = captured.FilePath, "replacement", true
	v.File = RemoteFile{Content: "replacement", Mtime: captured.File.Mtime}
	stashResource(v)
	if got := s.EditBuffer(captured, func(string) FileDraftBuffer { return b }); got != nil {
		t.Fatal("closed resource attached to same-path replacement")
	}
	fresh := s.Snapshot()
	v.RemoteIdentity = "different-host"
	if got := s.EditBuffer(fresh, func(string) FileDraftBuffer { return b }); got != nil {
		t.Fatal("captured source attached to a different remote identity")
	}
	if s.Snapshot().Draft != "replacement" {
		t.Fatal("rejected attachment mutated replacement")
	}
}

func TestRemoteExplicitDiscardReplacesCommittedBuffer(t *testing.T) {
	s, b := seededDraftFiles(t, "http://unused.invalid")
	b.ReplaceText("unsaved")
	s.Discard()
	if s.HasDirty() || b.CommittedText() != "base" {
		t.Fatal("discard left native document dirty")
	}
	s.SetDraft("explicit replacement")
	if b.CommittedText() != "explicit replacement" || s.Snapshot().Draft != "explicit replacement" {
		t.Fatal("model replacement did not update same native resource")
	}
}

func TestCloseSaveCapturesNativeTextAndRetainsNewInputAfterAcknowledgement(t *testing.T) {
	requests := make(chan map[string]any, 1)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dsh-remote/write" {
			t.Errorf("unexpected request %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		requests <- body
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "mtime": 11})
	}))
	defer srv.Close()
	defer unblock()
	s, b := seededDraftFiles(t, srv.URL)
	b.ReplaceText("captured native text")
	if err := s.SaveDirtyForClose(); err != nil {
		t.Fatal(err)
	}
	select {
	case body := <-requests:
		if body["content"] != "captured native text" || body["sessionId"] != "session-a" || body["path"] != "/work/a.txt" || body["expectedMtime"] != float64(10) {
			t.Fatal("write did not preserve captured owner/content/mtime", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("explicit close save never queued its write")
	}
	b.ReplaceText("new native text while saving")
	unblock()
	deadline := time.Now().Add(2 * time.Second)
	for s.Snapshot().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.Snapshot().Busy {
		t.Fatal("write acknowledgement did not settle")
	}
	if done, err := s.CheckDirtyForClose(); done || err == nil {
		t.Fatal("new native input was acknowledged as saved", done, err)
	}
	got := s.Snapshot()
	if got.File.Content != "captured native text" || got.Draft != "new native text while saving" || !s.HasDirty() {
		t.Fatal("acknowledgement discarded later committed input", got)
	}
	if s.closeSave != nil {
		t.Fatal("failed close attempt stranded its receipt cursor")
	}
	select {
	case body := <-requests:
		t.Fatal("close failure retried without a gesture", body)
	default:
	}
}
