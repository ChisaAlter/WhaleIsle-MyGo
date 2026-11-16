package projectctl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/projects"
)

type mockVisualSurface struct {
	mu       sync.Mutex
	height   atomic.Int32
	done     chan struct{}
	closed   atomic.Bool
	onExpire func()
	onHeight func()
}

func newMockVisualSurface(initialHeight int) *mockVisualSurface {
	s := &mockVisualSurface{done: make(chan struct{})}
	s.height.Store(int32(initialHeight))
	return s
}

func (s *mockVisualSurface) Present(x, y, width, height, clipX, clipY, clipWidth, clipHeight int, appearance string, variables map[string]string) error {
	return nil
}

func (s *mockVisualSurface) Done() <-chan struct{} { return s.done }

func (s *mockVisualSurface) Close() error {
	if s.closed.CompareAndSwap(false, true) {
		close(s.done)
	}
	return nil
}

func (s *mockVisualSurface) ContentHeight() int { return int(s.height.Load()) }

func (s *mockVisualSurface) SetHooks(onExpire, onHeight func()) {
	s.mu.Lock()
	s.onExpire = onExpire
	s.onHeight = onHeight
	s.mu.Unlock()
}

func (s *mockVisualSurface) triggerExpire() {
	s.mu.Lock()
	cb := s.onExpire
	s.mu.Unlock()
	if cb != nil {
		cb()
	}
}

func (s *mockVisualSurface) triggerHeight(h int) {
	s.height.Store(int32(h))
	s.mu.Lock()
	cb := s.onHeight
	s.mu.Unlock()
	if cb != nil {
		cb()
	}
}

func visualServerFixture(t *testing.T, htmlContent string) (*httptest.Server, *dsh.Client) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/api/session/readVisualReply":
			value = map[string]any{
				"attachment": map[string]any{
					"attachmentId": "att-A",
					"name":         "owned.html",
					"bytes":        len(htmlContent),
				},
				"html": htmlContent,
			}
		default:
			value = map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "server-response",
			"rpcId":  req.RPCID,
			"result": map[string]any{"ok": true, "value": value},
		})
	}))
	t.Cleanup(srv.Close)
	return srv, &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: srv.Client()}
}

func setupVisualController(t *testing.T, client *dsh.Client, htmlBytes int, deps HostDeps) (*Manager, *Controller, projectflow.DeliveryPageReference) {
	t.Helper()
	deps.HostSnapshot = func() HostSnapshot { return HostSnapshot{Client: client} }
	if deps.UpdateWindow == nil {
		deps.UpdateWindow = func(_ any, fn func()) {
			if fn != nil {
				fn()
			}
		}
	}
	if deps.VisualReaderAvailable == nil {
		deps.VisualReaderAvailable = func() (bool, string) { return true, "" }
	}
	mgr := NewManager(deps)
	c := mgr.Primary()

	var card projectflow.RequestCard
	raw := `{"id":"A","projectId":"owned-project","revision":3,"title":"A","status":"completed","delivery":{"id":"delivery-A","outcome":"completed","pages":[{"sessionId":"session-A","callId":"call-A","title":"Owned page","height":360,"file":{"attachmentId":"att-A","name":"owned.html","bytes":0}}]}}`
	if err := json.Unmarshal([]byte(raw), &card); err != nil {
		t.Fatal(err)
	}
	card.Delivery.Pages[0].File.Bytes = int64(htmlBytes)

	mgr.mu.Lock()
	c.flow = flowState{
		client:           client,
		expanded:         true,
		navigation:       1,
		generation:       1,
		requestProjectID: "owned-project",
		listing:          projectflow.Listing{Available: true, Projects: []projectflow.Project{{ID: "owned-project", Lifecycle: "ready"}}},
		requestCard:      &card,
	}
	c.rememberRequestCardLocked(&card)
	mgr.mu.Unlock()
	return mgr, c, card.Delivery.Pages[0]
}

func TestVisualInteractive250msLoopStopsOnCloseAndOnlyInvalidatesOnHeightOrReady(t *testing.T) {
	const html = "<html><body>ok</body></html>"
	_, client := visualServerFixture(t, html)
	surface := newMockVisualSurface(200)
	var invalidations atomic.Int32
	surfaceOpened := make(chan struct{}, 1)

	mgr, c, ref := setupVisualController(t, client, len(html), HostDeps{
		Invalidate: func(_ any) { invalidations.Add(1) },
		NewVisualSurface: func(_ context.Context, _ any, opts VisualSurfaceOptions) (VisualSurface, error) {
			select {
			case surfaceOpened <- struct{}{}:
			default:
			}
			return surface, nil
		},
	})

	var model projects.Model
	c.bindVisualResults("win", &model)
	_ = model.VisualPage("request:A", ref, 400)

	select {
	case <-surfaceOpened:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for NewVisualSurface")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		p := model.VisualPage("request:A", ref, 400)
		if !p.Loading && p.Surface != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	readyCount := invalidations.Load()
	if readyCount != 1 {
		t.Fatalf("expected exactly 1 readiness Invalidate, got %d", readyCount)
	}

	// Wait across multiple 250ms ticks without changing ContentHeight: must not invalidate.
	time.Sleep(600 * time.Millisecond)
	if got := invalidations.Load(); got != readyCount {
		t.Fatalf("expected 0 idle invalidations while height unchanged, got %d (was %d)", got, readyCount)
	}

	// Changing ContentHeight without calling hook must trigger exactly 1 Invalidate on tick.
	surface.height.Store(340)
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if invalidations.Load() == readyCount+1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := invalidations.Load(); got != readyCount+1 {
		t.Fatalf("expected 1 invalidation after height change, got %d", got)
	}

	// Changing height via hook must trigger 1 Invalidate and not duplicate on subsequent tick.
	surface.triggerHeight(420)
	time.Sleep(350 * time.Millisecond)
	if got := invalidations.Load(); got != readyCount+2 {
		t.Fatalf("expected 1 invalidation after hook height change without tick duplicate, got %d", got)
	}

	// Closing Manager must retire the surface and stop the 250ms loop cleanly.
	mgr.Close()
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if surface.closed.Load() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !surface.closed.Load() {
		t.Fatal("Manager.Close did not close interactive visual surface")
	}
}

func TestVisualInteractiveFallbackToStaticCaptureOnUnexpectedClose(t *testing.T) {
	const html = "<html><body>fallback</body></html>"
	_, client := visualServerFixture(t, html)
	surface := newMockVisualSurface(200)
	captureCalled := make(chan struct{}, 1)
	decoded := make(chan struct{}, 1)

	mgr, c, ref := setupVisualController(t, client, len(html), HostDeps{
		NewVisualSurface: func(_ context.Context, _ any, opts VisualSurfaceOptions) (VisualSurface, error) {
			return surface, nil
		},
		CaptureVisualHTML: func(_ context.Context, html string, width int, appearance string) (VisualCaptureResult, error) {
			select {
			case captureCalled <- struct{}{}:
			default:
			}
			return VisualCaptureResult{
				PNG:            base64.StdEncoding.EncodeToString([]byte("png-bytes")),
				ContentHeight:  500,
				CapturedHeight: 500,
			}, nil
		},
		DecodeVisualBitmap: func(preview *projects.VisualPagePreview, png []byte) error {
			if string(png) != "png-bytes" {
				t.Errorf("unexpected PNG payload: %q", png)
			}
			select {
			case decoded <- struct{}{}:
			default:
			}
			return nil
		},
	})
	defer mgr.Close()

	var model projects.Model
	c.bindVisualResults("win", &model)
	_ = model.VisualPage("request:A", ref, 400)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		p := model.VisualPage("request:A", ref, 400)
		if p.Surface != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Simulate unexpected surface termination while owner is still current.
	_ = surface.Close()

	select {
	case <-captureCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for static fallback CaptureVisualHTML")
	}
	select {
	case <-decoded:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DecodeVisualBitmap")
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		p := model.VisualPage("request:A", ref, 400)
		if !p.Loading && p.Surface == nil && p.InteractiveError != "" && p.ContentHeight == 500 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("fallback static capture measurements were not published")
}

func TestVisualInteractiveRetireOnExpireHook(t *testing.T) {
	const html = "<html><body>expire</body></html>"
	_, client := visualServerFixture(t, html)
	surface := newMockVisualSurface(200)

	mgr, c, ref := setupVisualController(t, client, len(html), HostDeps{
		NewVisualSurface: func(_ context.Context, _ any, opts VisualSurfaceOptions) (VisualSurface, error) {
			return surface, nil
		},
	})
	defer mgr.Close()

	var model projects.Model
	c.bindVisualResults("win", &model)
	_ = model.VisualPage("request:A", ref, 400)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		p := model.VisualPage("request:A", ref, 400)
		if p.Surface != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Change navigation so owner becomes stale, then fire onExpire immediately without waiting for 250ms tick.
	mgr.mu.Lock()
	c.flow.navigation++
	mgr.mu.Unlock()
	surface.triggerExpire()

	deadline = time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline) {
		if surface.closed.Load() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("onExpire hook did not immediately retire stale visual surface")
}
