package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type interactiveFixtureHandle struct {
	done chan struct{}
	once sync.Once
}

func (h *interactiveFixtureHandle) Done() <-chan struct{} { return h.done }
func (h *interactiveFixtureHandle) Close() error {
	h.once.Do(func() { close(h.done) })
	return nil
}

func interactiveFixtureStore(t *testing.T) (*Store, string, <-chan struct{}) {
	t.Helper()
	// Deliberately contains interactive source; this test only transports bytes.
	html := `<form onsubmit="event.preventDefault();out.textContent=input.value"><input id="input"><button>Apply</button><output id="out"></output></form><div style="height:9000px">bottom</div>`
	ref := VisualReplyFile{AttachmentID: "page", Name: "page.html", Bytes: int64(len(html))}
	raw, _ := json.Marshal(map[string]any{"type": "tool/result", "surfaceOp": "append", "data": map[string]any{"message": map[string]any{"content": []any{map[string]any{"type": "file", "attachment": ref}}}, "meta": map[string]any{"visualReply": map[string]any{"version": 1, "title": "Fixture"}}}})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/session/readVisualReply" {
			t.Errorf("unexpected read: %s", r.URL.Path)
			http.Error(w, "unexpected", 400)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcOK(w, body["rpcId"], map[string]any{"attachment": ref, "html": html})
	}))
	t.Cleanup(srv.Close)
	s := New()
	changed := make(chan struct{}, 16)
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	s.selectedID = "A"
	s.messages = []Message{{Raw: raw}}
	t.Cleanup(s.CloseVisualReply)
	return s, html, changed
}

func waitInteractiveFixture(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("interactive fixture did not settle")
	}
}

// No renderer, model page, external URL or OS helper runs in these fixtures.
func TestInteractiveVisualReplyAuthorizedLifetime(t *testing.T) {
	s, source, changed := interactiveFixtureStore(t)
	calls := 0
	fail := func(context.Context, string) (VisualReplyInteractive, error) {
		calls++
		return nil, errors.New("containment unavailable")
	}
	s.OpenInteractiveVisualReply(context.Background(), "B", "page", fail)
	s.OpenInteractiveVisualReply(context.Background(), "A", "unknown", fail)
	if calls != 0 {
		t.Fatal("unpublished or cross-session page reached the opener")
	}
	s.OpenInteractiveVisualReply(context.Background(), "A", "page", fail)
	state := s.VisualReplySnapshot()
	if calls != 1 || !state.Ready || state.HTML != source || state.InteractiveActive || state.InteractiveError != "containment unavailable" {
		t.Fatalf("failed launch did not preserve authorized source: %+v", state)
	}

	handle := &interactiveFixtureHandle{done: make(chan struct{})}
	finished := make(chan struct{})
	go func() {
		s.OpenInteractiveVisualReply(context.Background(), "A", "page", func(_ context.Context, html string) (VisualReplyInteractive, error) {
			if html != source {
				t.Error("interactive source was not the exact authorized attachment")
			}
			return handle, nil
		})
		close(finished)
	}()
	deadline := time.After(3 * time.Second)
	for !s.VisualReplySnapshot().InteractiveActive {
		select {
		case <-changed:
		case <-deadline:
			t.Fatal("interactive handle never became active")
		}
	}
	if s.VisualReplySnapshot().PreviewID != 0 {
		t.Fatal("interactive launch substituted a PNG")
	}
	s.StopInteractiveVisualReply()
	waitInteractiveFixture(t, finished)
	waitInteractiveFixture(t, handle.Done())
	if state = s.VisualReplySnapshot(); !state.Ready || state.HTML != source || state.InteractiveActive || state.InteractiveLoading {
		t.Fatal("stop lost source or kept interaction active", state)
	}
}

func TestInteractiveVisualReplyLateLaunchRevocation(t *testing.T) {
	for _, revoke := range []string{"close", "host", "session"} {
		t.Run(revoke, func(t *testing.T) {
			s, _, _ := interactiveFixtureStore(t)
			entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			handle := &interactiveFixtureHandle{done: make(chan struct{})}
			go func() {
				s.OpenInteractiveVisualReply(context.Background(), "A", "page", func(ctx context.Context, _ string) (VisualReplyInteractive, error) {
					close(entered)
					<-release // Simulate an opener finishing after cancellation.
					if ctx.Err() == nil {
						t.Error("revocation did not cancel the helper context")
					}
					return handle, nil
				})
				close(finished)
			}()
			waitInteractiveFixture(t, entered)
			switch revoke {
			case "close":
				s.CloseVisualReply()
			case "host":
				s.Attach(nil, nil)
			case "session":
				s.mu.Lock()
				s.selectSessionLocked("B")
				s.mu.Unlock()
			}
			close(release)
			waitInteractiveFixture(t, finished)
			waitInteractiveFixture(t, handle.Done())
			if state := s.VisualReplySnapshot(); state.Ready || state.InteractiveActive || state.InteractiveLoading {
				t.Fatal("late helper was published after revocation", state)
			}
		})
	}
}
