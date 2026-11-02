package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type startupReceiptHost struct {
	s                                       *Store
	client                                  *dsh.Client
	mu                                      sync.Mutex
	listError                               bool
	initDeclined                            bool
	rows                                    []map[string]any
	initRelease, listRelease, createRelease <-chan struct{}
	initStarted, listStarted, createStarted chan struct{}
	notified                                chan struct{}
	calls                                   map[string]int
}

func newStartupReceiptHost(t *testing.T) *startupReceiptHost {
	t.Helper()
	h := &startupReceiptHost{s: New(), notified: make(chan struct{}, 64), initStarted: make(chan struct{}, 8), listStarted: make(chan struct{}, 8), createStarted: make(chan struct{}, 8), calls: map[string]int{}}
	h.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://startup-fixture.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			RPCID string `json:"rpcId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, err
		}
		method := strings.TrimPrefix(r.URL.Path, "/api/")
		h.mu.Lock()
		h.calls[method]++
		failed, declined, rows := h.listError, h.initDeclined, append([]map[string]any(nil), h.rows...)
		initRelease, listRelease, createRelease := h.initRelease, h.listRelease, h.createRelease
		h.mu.Unlock()
		var value any = map[string]any{}
		switch method {
		case "workspace/initializeDefault":
			h.initStarted <- struct{}{}
			if initRelease != nil {
				<-initRelease
			}
			value = map[string]any{"workspace": map[string]any{"workspaceId": "default", "path": "/fresh", "sessionIds": []string{}}}
			if declined {
				value = nil
			}
		case "session/list":
			h.listStarted <- struct{}{}
			if listRelease != nil {
				<-listRelease
			}
			if failed {
				return nil, errors.New("list unavailable")
			}
			value = map[string]any{"items": rows}
		case "session/create":
			h.createStarted <- struct{}{}
			if createRelease != nil {
				<-createRelease
			}
			value = map[string]any{"sessionId": "startup-blank"}
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	h.s.Attach(h.client, func() {
		select {
		case h.notified <- struct{}{}:
		default:
		}
	})
	h.s.AttachMux(&dsh.Mux{Endpoint: h.client.Endpoint})
	h.s.SetWorkspaceCwd("/fresh")
	return h
}

func startupSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatal("startup fixture barrier timed out")
	}
}
func (h *startupReceiptHost) wait(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for !condition() {
		select {
		case <-h.notified:
		case <-deadline:
			t.Fatal("startup state did not settle")
		}
	}
}
func (h *startupReceiptHost) baseline() {
	h.s.applyWorkspaceBaseline(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"default","path":"/fresh","sessionIds":[]}],"scratchCwd":"/scratch","pinnedSessionIds":[],"archivedSessionIds":[]}}`))
}
func (h *startupReceiptHost) initialized(t *testing.T) {
	h.s.InitializeDefaultWorkspace(context.Background())
	h.wait(t, func() bool {
		h.s.mu.Lock()
		defer h.s.mu.Unlock()
		return h.s.initialWorkspaceSession.accepted && !h.s.listLoading && h.s.listAuthoritativeHost == h.s.eventsGeneration
	})
}

func TestInitialWorkspaceSessionWaitsForBothAuthoritativeBaselines(t *testing.T) {
	h := newStartupReceiptHost(t)
	initRelease, listRelease := make(chan struct{}), make(chan struct{})
	h.initRelease, h.listRelease = initRelease, listRelease
	h.s.InitializeDefaultWorkspace(context.Background())
	startupSignal(t, h.initStarted)
	if h.s.InitialWorkspaceSession().Ready {
		t.Fatal("ready before init")
	}
	close(initRelease)
	startupSignal(t, h.listStarted)
	h.baseline()
	if h.s.InitialWorkspaceSession().Ready {
		t.Fatal("loading list treated as empty")
	}
	close(listRelease)
	h.wait(t, func() bool { return h.s.InitialWorkspaceSession().Ready })
	r := h.s.InitialWorkspaceSession()
	if r.WorkspaceID != "default" || r.Binding == 0 || r.HostGeneration != h.s.Snapshot().WorkspaceLayout.HostGeneration {
		t.Fatalf("bad initial receipt: %+v", r)
	}
	target, err := h.s.CaptureWorkspaceSessionTarget(r.HostGeneration, r.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	id, err := target.Create(context.Background())
	if err != nil || id != "startup-blank" {
		t.Fatalf("create: %q %v", id, err)
	}
	if h.s.SelectedSessionID() != "" || h.s.InitialWorkspaceSession().Ready {
		t.Fatal("create selected or nonempty roster remained ready")
	}
}

func TestInitialWorkspaceSessionRejectsFailureExistingRowsAndNavigation(t *testing.T) {
	for _, change := range []string{"failed-list", "registry-not-ready", "registry-error", "missing-default", "different-path", "existing-row", "existing-member", "selected", "navigation", "selection-away-back"} {
		t.Run(change, func(t *testing.T) {
			h := newStartupReceiptHost(t)
			h.baseline()
			if change == "failed-list" {
				h.listError = true
			}
			if change == "existing-row" {
				h.rows = []map[string]any{{"sessionId": "old", "cwd": "/fresh", "blank": true}}
			}
			h.s.InitializeDefaultWorkspace(context.Background())
			h.wait(t, func() bool {
				h.s.mu.Lock()
				defer h.s.mu.Unlock()
				return h.s.initialWorkspaceSession.accepted && !h.s.listLoading && (h.s.listErr != nil || h.s.listAuthoritativeHost == h.s.eventsGeneration)
			})
			h.s.mu.Lock()
			switch change {
			case "registry-not-ready":
				h.s.workspaceReady = false
			case "registry-error":
				h.s.workspaceErr = errors.New("registry unavailable")
			case "missing-default":
				delete(h.s.workspaceMemberships, "default")
			case "different-path":
				ws := h.s.workspaceMemberships["default"]
				ws.Path = "/other"
				h.s.workspaceMemberships["default"] = ws
			case "existing-member":
				ws := h.s.workspaceMemberships["default"]
				ws.SessionIDs = []string{"not-in-list"}
				h.s.workspaceMemberships["default"] = ws
			case "selected":
				h.s.selectSessionLocked("old")
			case "navigation":
				h.s.invalidateWorkspaceNavigationLocked()
			case "selection-away-back":
				h.s.selectSessionLocked("other")
				h.s.selectSessionLocked("")
			}
			h.s.mu.Unlock()
			if h.s.InitialWorkspaceSession().Ready {
				t.Fatal("unsafe initial opportunity")
			}
			if change == "existing-row" && h.s.SelectedSessionID() != "old" {
				t.Fatal("legacy first-row selection changed")
			}
		})
	}
}

func TestInitialWorkspaceSessionStaleInitAndReconnectDoNotRearm(t *testing.T) {
	h := newStartupReceiptHost(t)
	release := make(chan struct{})
	h.initRelease = release
	h.s.InitializeDefaultWorkspace(context.Background())
	startupSignal(t, h.initStarted)
	first := h.s.InitialWorkspaceSession()
	h.s.BeginEvents(context.Background(), h.client)
	close(release)
	h.mu.Lock()
	h.initRelease = nil
	h.mu.Unlock()
	h.s.InitializeDefaultWorkspace(context.Background())
	h.wait(t, func() bool {
		h.s.mu.Lock()
		defer h.s.mu.Unlock()
		return h.s.initialWorkspaceSession.accepted && !h.s.listLoading && h.s.listAuthoritativeHost == h.s.eventsGeneration
	})
	h.baseline()
	r := h.s.InitialWorkspaceSession()
	if !r.Ready || r.Binding != first.Binding || r.HostGeneration == first.HostGeneration {
		t.Fatalf("reconnect lost binding or current authority: first=%+v now=%+v", first, r)
	}
	h.s.mu.Lock()
	h.s.selectSessionLocked("opened")
	h.s.selectSessionLocked("")
	h.s.mu.Unlock()
	h.s.BeginEvents(context.Background(), h.client)
	h.baseline()
	h.s.RefreshList(context.Background())
	h.wait(t, func() bool {
		h.s.mu.Lock()
		defer h.s.mu.Unlock()
		return !h.s.listLoading && h.s.listAuthoritativeHost == h.s.eventsGeneration
	})
	if h.s.InitialWorkspaceSession().Ready || h.s.InitialWorkspaceSession().Binding != first.Binding {
		t.Fatal("reconnect rearmed after navigation")
	}
	h.s.Detach()
	h.s.Attach(h.client, nil)
	if h.s.InitialWorkspaceSession().Binding == first.Binding {
		t.Fatal("new Host binding was not reset")
	}
}

func TestInitialWorkspaceSessionHonorsHostDeclinedDefault(t *testing.T) {
	h := newStartupReceiptHost(t)
	h.initDeclined = true
	h.s.applyWorkspaceBaseline(json.RawMessage(`{"type":"baseline","value":{"items":[{"workspaceId":"registered","path":"/fresh","sessionIds":[]}],"scratchCwd":"/scratch","pinnedSessionIds":[],"archivedSessionIds":[]}}`))
	h.s.InitializeDefaultWorkspace(context.Background())
	h.wait(t, func() bool {
		h.s.mu.Lock()
		defer h.s.mu.Unlock()
		return h.s.initialWorkspaceSession.completed && !h.s.listLoading && h.s.listAuthoritativeHost == h.s.eventsGeneration
	})
	h.s.mu.Lock()
	accepted := h.s.initialWorkspaceSession.accepted
	h.s.mu.Unlock()
	if accepted || h.s.InitialWorkspaceSession().Ready || h.s.WorkspaceID() != "registered" || h.s.SelectedSessionID() != "" {
		t.Fatal("Host-declined initialization was treated as an accepted default")
	}
	h.s.BeginEvents(context.Background(), h.client)
	h.s.InitializeDefaultWorkspace(context.Background())
	h.mu.Lock()
	calls := h.calls["workspace/initializeDefault"]
	h.mu.Unlock()
	if calls != 1 {
		t.Fatalf("Host decline rearmed on reconnect: %d calls", calls)
	}
}

func TestInitialWorkspaceSessionDelayedCreateCannotRetargetLaterNavigation(t *testing.T) {
	h := newStartupReceiptHost(t)
	h.baseline()
	h.initialized(t)
	r := h.s.InitialWorkspaceSession()
	target, err := h.s.CaptureWorkspaceSessionTarget(r.HostGeneration, r.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	h.createRelease = release
	done := make(chan error, 1)
	go func() { _, err := target.Create(context.Background()); done <- err }()
	startupSignal(t, h.createStarted)
	h.s.mu.Lock()
	h.s.invalidateWorkspaceNavigationLocked()
	h.s.mu.Unlock()
	close(release)
	if err := <-done; err == nil {
		t.Fatal("stale startup creation succeeded")
	}
	if target.Select("startup-blank") || h.s.SelectedSessionID() != "" {
		t.Fatal("late startup switched navigation")
	}
}
