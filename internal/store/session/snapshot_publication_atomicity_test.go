package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type snapshotPublicationGate struct {
	point   string
	reached chan struct{}
	release chan struct{}
	once    sync.Once
}

func (g *snapshotPublicationGate) unblock() { g.once.Do(func() { close(g.release) }) }

func gateSnapshotPublications(t *testing.T, gates ...*snapshotPublicationGate) {
	t.Helper()
	hook := func(point string) {
		for _, gate := range gates {
			if gate.point == point {
				gate.reached <- struct{}{}
				<-gate.release
			}
		}
	}
	if !snapshotNotificationHook.CompareAndSwap(nil, &hook) {
		t.Fatal("snapshot publication hook already installed")
	}
	t.Cleanup(func() {
		snapshotNotificationHook.Store(nil)
		for _, gate := range gates {
			gate.unblock()
		}
	})
}

func newSnapshotPublicationGate(point string) *snapshotPublicationGate {
	return &snapshotPublicationGate{point: point, reached: make(chan struct{}, 1), release: make(chan struct{})}
}

func waitSnapshotPublication(t *testing.T, ch <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func TestForkSnapshotBeforeNotification(t *testing.T) {
	for _, reply := range []string{"success", "error", "empty"} {
		t.Run(reply, func(t *testing.T) {
			start, complete := newSnapshotPublicationGate("fork-start"), newSnapshotPublicationGate("fork-complete")
			gateSnapshotPublications(t, start, complete)
			requestReached, releaseRequest := make(chan struct{}), make(chan struct{})
			var requestOnce sync.Once
			release := func() { requestOnce.Do(func() { close(releaseRequest) }) }
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if strings.HasSuffix(r.URL.Path, "/session/fork") {
					close(requestReached)
					<-releaseRequest
					if reply == "error" {
						http.Error(w, "fork failed", http.StatusInternalServerError)
						return
					}
					id := "child"
					if reply == "empty" {
						id = ""
					}
					rpcOK(w, body["rpcId"], map[string]any{"sessionId": id})
					return
				}
				rpcOK(w, body["rpcId"], map[string]any{})
			}))
			defer server.Close()
			defer release()
			s := New()
			s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
			s.mu.Lock()
			s.selectSessionLocked("source")
			s.messages = branchFixture()
			s.mu.Unlock()
			frame := s.BeginFrame()
			defer frame.EndFrame()
			if s.Snapshot().HistoryBranchBusy || frame.Snapshot().HistoryBranchBusy {
				t.Fatal("fork fixture unexpectedly busy")
			}
			returned := make(chan struct{})
			go func() { s.ForkTurn(context.Background(), "source", 9); close(returned) }()
			waitSnapshotPublication(t, start.reached, "fork start publication")
			if !s.Snapshot().HistoryBranchBusy || !frame.Snapshot().HistoryBranchBusy {
				t.Error("fork start must be visible before fire")
			}
			start.unblock()
			waitSnapshotPublication(t, returned, "fork admission")
			waitSnapshotPublication(t, requestReached, "fork request")
			_ = s.Snapshot()
			_ = frame.Snapshot()
			release()
			waitSnapshotPublication(t, complete.reached, "fork completion publication")
			for _, got := range []ViewState{s.Snapshot(), frame.Snapshot()} {
				if got.HistoryBranchBusy {
					t.Error("fork completion remained busy before fire")
				}
				if reply == "success" && (got.SelectedID != "child" || got.LastAction != "session forked") {
					t.Errorf("fork completion state: selected=%q action=%q", got.SelectedID, got.LastAction)
				} else if reply != "success" && (got.SelectedID != "source" || !strings.HasPrefix(got.LastAction, "session/fork:")) {
					t.Errorf("fork failure state: selected=%q action=%q", got.SelectedID, got.LastAction)
				}
			}
			complete.unblock()
		})
	}
}

func TestPermissionSnapshotBeforeNotification(t *testing.T) {
	gate := newSnapshotPublicationGate("permission-settled")
	gateSnapshotPublications(t, gate)
	projectionReached, releaseProjection := make(chan struct{}), make(chan struct{})
	var projectionOnce sync.Once
	release := func() { projectionOnce.Do(func() { close(releaseProjection) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/commands/execute"):
			rpcOK(w, body["rpcId"], map[string]any{"result": map[string]any{"kind": "success"}})
		case strings.HasSuffix(r.URL.Path, "/session/projections"):
			close(projectionReached)
			<-releaseProjection
			rpcOK(w, body["rpcId"], map[string]any{"asOfSeq": 2, "values": map[string]any{"permissions": map[string]any{"currentValue": "workspace-write"}}})
		default:
			http.Error(w, "unexpected RPC", http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	defer release()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	s.mu.Lock()
	s.selectSessionLocked("A")
	s.permissionOptions = []PermissionOption{{Label: "Read", Value: "read-only"}, {Label: "Write", Value: "workspace-write"}}
	s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"read-only"}}`), 1)
	s.mu.Unlock()
	frame := s.BeginFrame()
	defer frame.EndFrame()
	if got := s.Snapshot().SelectedPermission; got != "Read" {
		t.Fatalf("initial permission = %q", got)
	}
	_ = frame.Snapshot()
	s.SelectPermissionValueForSession(context.Background(), "A", "workspace-write")
	waitSnapshotPublication(t, projectionReached, "permission projection request")
	if got := s.Snapshot().SelectedPermission; got != "Read" {
		t.Errorf("unconfirmed permission = %q", got)
	}
	_ = frame.Snapshot()
	release()
	waitSnapshotPublication(t, gate.reached, "permission settlement publication")
	for _, got := range []ViewState{s.Snapshot(), frame.Snapshot()} {
		if got.SelectedPermission != "Write" || got.PermissionsErr != "" {
			t.Errorf("permission snapshot before fire: selected=%q error=%q", got.SelectedPermission, got.PermissionsErr)
		}
	}
	gate.unblock()
}

func TestPermissionCatalogSnapshotBeforeNotification(t *testing.T) {
	gate := newSnapshotPublicationGate("permission-catalog")
	gateSnapshotPublications(t, gate)
	requestReached, releaseRequest := make(chan struct{}), make(chan struct{})
	var requestOnce sync.Once
	release := func() { requestOnce.Do(func() { close(releaseRequest) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !strings.HasSuffix(r.URL.Path, "/permissionPresets/catalog") {
			http.Error(w, "unexpected RPC", http.StatusInternalServerError)
			return
		}
		close(requestReached)
		<-releaseRequest
		rpcOK(w, body["rpcId"], map[string]any{"defaultOptions": []any{}, "options": []any{map[string]any{"value": "read-only", "name": "Read revised"}, map[string]any{"value": "workspace-write", "name": "Write"}}})
	}))
	defer server.Close()
	defer release()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	s.mu.Lock()
	s.selectSessionLocked("A")
	s.permissionOptions = []PermissionOption{{Label: "Read", Value: "read-only"}}
	s.applyPermissionValuesLocked(json.RawMessage(`{"permissions":{"currentValue":"read-only"}}`), 1)
	s.mu.Unlock()
	frame := s.BeginFrame()
	defer frame.EndFrame()
	s.RefreshPermissionCatalog(context.Background())
	waitSnapshotPublication(t, requestReached, "permission catalog request")
	if got := s.Snapshot().SelectedPermission; got != "Read" {
		t.Fatalf("initial permission = %q", got)
	}
	_ = frame.Snapshot()
	release()
	waitSnapshotPublication(t, gate.reached, "permission catalog publication")
	for _, got := range []ViewState{s.Snapshot(), frame.Snapshot()} {
		if got.SelectedPermission != "Read revised" || !slices.Equal(got.PermissionLabels, []string{"Read revised", "Write"}) {
			t.Errorf("permission catalog snapshot before fire: selected=%q labels=%v", got.SelectedPermission, got.PermissionLabels)
		}
	}
	gate.unblock()
}

func TestCommandCatalogSnapshotBeforeNotification(t *testing.T) {
	start, complete := newSnapshotPublicationGate("command-start"), newSnapshotPublicationGate("command-complete")
	gateSnapshotPublications(t, start, complete)
	requestReached, releaseRequest := make(chan struct{}), make(chan struct{})
	var requestOnce sync.Once
	release := func() { requestOnce.Do(func() { close(releaseRequest) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		close(requestReached)
		<-releaseRequest
		rpcOK(w, body["rpcId"], []any{map[string]any{"name": "inspect", "description": "Inspect"}})
	}))
	defer server.Close()
	defer release()
	s := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}
	s.Attach(client, nil)
	s.mu.Lock()
	s.selectSessionLocked("A")
	s.commandCatalog.client, s.commandCatalog.sessionID = client, "A"
	s.mu.Unlock()
	frame := s.BeginFrame()
	defer frame.EndFrame()
	_ = s.Snapshot()
	_ = frame.Snapshot()
	returned := make(chan struct{})
	go func() { s.EnsureSlashCommands(context.Background()); close(returned) }()
	waitSnapshotPublication(t, start.reached, "command loading publication")
	for _, got := range []ViewState{s.Snapshot(), frame.Snapshot()} {
		if !got.SlashCommandsLoading || got.SlashCommandsKnown {
			t.Errorf("command loading before notification: loading=%v known=%v", got.SlashCommandsLoading, got.SlashCommandsKnown)
		}
	}
	start.unblock()
	waitSnapshotPublication(t, returned, "command load admission")
	waitSnapshotPublication(t, requestReached, "commands request")
	_ = s.Snapshot()
	_ = frame.Snapshot()
	release()
	waitSnapshotPublication(t, complete.reached, "command catalog publication")
	for _, got := range []ViewState{s.Snapshot(), frame.Snapshot()} {
		if got.SlashCommandsLoading || !got.SlashCommandsKnown || len(got.SlashCommands) != 1 || got.SlashCommands[0].Name != "inspect" {
			t.Errorf("command catalog before notification: loading=%v known=%v commands=%v", got.SlashCommandsLoading, got.SlashCommandsKnown, got.SlashCommands)
		}
	}
	complete.unblock()
}

func TestCommandEmitMissDoesNotInvalidateSnapshot(t *testing.T) {
	s := New()
	_ = s.Snapshot()
	builds, _ := s.SnapshotStats()
	s.mu.Lock()
	changed := s.applyCommandCatalogEmitLocked(dsh.EventsEmit{Event: "unrelated"})
	s.mu.Unlock()
	if changed {
		t.Fatal("unrelated event changed the command catalog")
	}
	_ = s.Snapshot()
	after, _ := s.SnapshotStats()
	if after != builds {
		t.Fatalf("unrelated command event rebuilt snapshot: %d -> %d", builds, after)
	}
}
