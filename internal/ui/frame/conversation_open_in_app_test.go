package frame

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

type headerOpenRequest struct {
	App     string `json:"app"`
	Path    string `json:"path"`
	release chan struct{}
}

type headerOpenTransport struct {
	apps          []string
	catalogStatus int
	catalogCalls  atomic.Int32
	opens         chan headerOpenRequest
}

// Every HTTP request is intercepted here, including unrelated selected-Session
// reads. No socket, Host, shell application, user setting or device is used.
func (f *headerOpenTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	status := http.StatusOK
	var body any = map[string]any{}
	switch r.URL.Path {
	case "/open-in-app/apps":
		f.catalogCalls.Add(1)
		body = map[string]any{"apps": f.apps}
		if f.catalogStatus != 0 {
			status = f.catalogStatus
		}
	case "/open-in-app/open":
		var q headerOpenRequest
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			return nil, err
		}
		q.release = make(chan struct{})
		f.opens <- q
		select {
		case <-q.release:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	case "/api/session/list":
		body = map[string]any{"items": []map[string]any{{"sessionId": "A", "cwd": "C:/fixture/A", "title": "A"}, {"sessionId": "B", "cwd": "C:/fixture/B", "title": "B"}}}
	default:
		if strings.HasPrefix(r.URL.Path, "/open-in-app/icon/") {
			status = http.StatusNotFound
		}
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		var wire struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&wire)
		body = map[string]any{"type": "server-response", "rpcId": wire.RPCID, "result": map[string]any{"ok": true, "value": body}}
	}
	data, _ := json.Marshal(body)
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(data))), Request: r}, nil
}

func waitHeaderOpen(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !check() {
		if time.Now().After(deadline) {
			t.Fatal("offline header operation did not settle")
		}
		time.Sleep(time.Millisecond)
	}
}

func newHeaderOpenStore(t *testing.T, transport *headerOpenTransport) *session.Store {
	t.Helper()
	store := session.New()
	store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://offline.invalid"}, HTTP: &http.Client{Transport: transport}}, nil)
	store.RefreshList(context.Background())
	waitHeaderOpen(t, func() bool { return len(store.Snapshot().Items) == 2 })
	store.Select("A")
	return store
}

func nextHeaderOpen(t *testing.T, f *headerOpenTransport) headerOpenRequest {
	t.Helper()
	select {
	case q := <-f.opens:
		return q
	case <-time.After(2 * time.Second):
		t.Fatal("offline launch was not admitted")
		return headerOpenRequest{}
	}
}

func TestWorkspaceOpenInAppCacheBusyChoiceAndSaveFailure(t *testing.T) {
	f := &headerOpenTransport{apps: []string{"unknown-app", "vscode", "explorer"}, opens: make(chan headerOpenRequest, 4)}
	store := newHeaderOpenStore(t, f)
	saved := make(chan string, 2)
	state := NewWorkspaceOpenInAppState("missing-app", func(id string) error { saved <- id; return errors.New("offline config write failure") })
	defer state.Close()
	state.ObserveHost(store)
	waitHeaderOpen(t, func() bool { state.mu.Lock(); defer state.mu.Unlock(); return len(state.apps) == 2 })
	mounted := true
	tt := ui.NewTester(func(c *ui.Context) {
		defer state.FinishFrame(c)
		if mounted {
			conversationOpenInApp(c, Model{Store: store, WorkspaceOpenInApp: state}, store.Snapshot(), theme.Resolve("deepseek.light").Light)
		}
	}, 600, 100)
	if _, ok := tt.Find("Open in VS Code"); !ok {
		if _, ok = tt.Find("用 VS Code 打开"); !ok {
			t.Fatal("missing preferred fallback button")
		}
	}
	if !state.OpenCurrent(store) {
		t.Fatal("default launch was not admitted")
	}
	first := nextHeaderOpen(t, f)
	if first.App != "vscode" || first.Path != "C:/fixture/A" {
		t.Fatalf("fallback/captured directory: %+v", first)
	}
	if state.OpenCurrent(store) || state.launch("A", "C:/fixture/A", "explorer", true) {
		t.Fatal("busy shared controller admitted a second gesture")
	}
	store.Select("B")
	close(first.release)
	waitHeaderOpen(t, func() bool { state.mu.Lock(); defer state.mu.Unlock(); return !state.busy })
	select {
	case id := <-saved:
		t.Fatalf("primary action remembered %s", id)
	default:
	}
	tt.Frame()
	if !state.launch("B", "C:/fixture/B", "explorer", true) {
		t.Fatal("explicit application launch was not admitted")
	}
	second := nextHeaderOpen(t, f)
	if second.App != "explorer" || second.Path != "C:/fixture/B" {
		t.Fatalf("menu launch: %+v", second)
	}
	close(second.release)
	waitHeaderOpen(t, func() bool { state.mu.Lock(); defer state.mu.Unlock(); return !state.busy })
	select {
	case id := <-saved:
		if id != "explorer" {
			t.Fatal("wrong remembered app")
		}
	default:
		t.Fatal("successful explicit action did not save")
	}
	state.mu.Lock()
	choice, failure := state.choice, state.failure
	state.mu.Unlock()
	if choice != "explorer" || failure != archiveText("已打开应用，但无法记住本次选择。", "Application opened, but the choice could not be saved.") {
		t.Fatalf("save failure misreported launch: %q %q", choice, failure)
	}
	tt.Frame()
	mounted = false
	tt.Frame()
	state.mu.Lock()
	cache, view, toast := len(state.apps), state.viewPath, state.toastID
	state.mu.Unlock()
	if cache != 2 || view != "" || toast != "" {
		t.Fatal("unmount retained feedback or discarded catalog")
	}
	for range 3 {
		state.ObserveHost(store)
	}
	if f.catalogCalls.Load() != 1 {
		t.Fatal("Session navigation/unmount reloaded availability")
	}
}

func TestWorkspaceOpenInAppHostChangeRetiresPendingLaunch(t *testing.T) {
	oldTransport := &headerOpenTransport{apps: []string{"vscode"}, opens: make(chan headerOpenRequest, 2)}
	store := newHeaderOpenStore(t, oldTransport)
	var saves atomic.Int32
	state := NewWorkspaceOpenInAppState("", func(string) error { saves.Add(1); return nil })
	defer state.Close()
	state.ObserveHost(store)
	waitHeaderOpen(t, func() bool { state.mu.Lock(); defer state.mu.Unlock(); return len(state.apps) == 1 })
	if !state.launch("A", "C:/fixture/A", "vscode", true) {
		t.Fatal("old launch not admitted")
	}
	old := nextHeaderOpen(t, oldTransport)
	nextTransport := &headerOpenTransport{apps: []string{"explorer"}, opens: make(chan headerOpenRequest, 2)}
	store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://next.invalid"}, HTTP: &http.Client{Transport: nextTransport}}, nil)
	state.ObserveHost(store)
	close(old.release)
	waitHeaderOpen(t, func() bool {
		state.mu.Lock()
		defer state.mu.Unlock()
		return len(state.apps) == 1 && state.apps[0].id == "explorer"
	})
	state.mu.Lock()
	busy, choice, failure := state.busy, state.choice, state.failure
	state.mu.Unlock()
	if busy || choice != "" || failure != "" || saves.Load() != 0 {
		t.Fatal("old Host completion leaked into new controller")
	}
}

func TestWorkspaceOpenInAppFailedCatalogStaysHiddenWithoutReload(t *testing.T) {
	f := &headerOpenTransport{catalogStatus: http.StatusServiceUnavailable, opens: make(chan headerOpenRequest, 1)}
	store := newHeaderOpenStore(t, f)
	state := NewWorkspaceOpenInAppState("vscode", nil)
	defer state.Close()
	state.ObserveHost(store)
	waitHeaderOpen(t, func() bool { return f.catalogCalls.Load() == 1 })
	store.Select("B")
	state.ObserveHost(store)
	if state.OpenCurrent(store) {
		t.Fatal("unavailable catalog launched a guessed default")
	}
	if f.catalogCalls.Load() != 1 {
		t.Fatal("failed once-per-Host availability silently retried")
	}
}
