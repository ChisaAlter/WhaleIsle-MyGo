package projectctl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
)

func lexicalTestValidator(root, canonical string) (string, error) {
	r := strings.ReplaceAll(strings.TrimSpace(root), `\`, "/")
	v := strings.ReplaceAll(strings.TrimSpace(canonical), `\`, "/")
	if r == "" || v == "" || strings.ContainsRune(v, 0) {
		return "", errors.New("invalid")
	}
	r = path.Clean(r)
	v = path.Clean(v)
	prefix := strings.TrimRight(r, "/") + "/"
	if !strings.HasPrefix(strings.ToLower(v), strings.ToLower(prefix)) {
		return "", errors.New("outside")
	}
	rel := v[len(prefix):]
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", errors.New("escape")
	}
	return rel, nil
}

func sessionStoreFixture(t *testing.T, items []map[string]any, selected string) (*session.Store, *dsh.Client) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/api/session/list":
			value = map[string]any{"items": items}
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

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: srv.Client()}
	store := session.New()
	store.Attach(client, nil)
	t.Cleanup(store.Detach)
	store.RefreshList(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(store.Snapshot().Items) == len(items) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if selected != "" {
		store.Select(selected)
	}
	return store, client
}

func TestTargetRelativeAndFailClosed(t *testing.T) {
	// Nil validator must fail closed.
	if _, err := TargetRelative(projectflow.FileTarget{ProjectID: "p", CoordinatorSessionID: "c", Root: "/a", CanonicalPath: "/a/b.txt"}, nil); err == nil {
		t.Fatal("nil validator admitted target")
	}

	for _, tc := range []struct {
		root, p, want string
		valid         bool
	}{
		{"/project/actual", "/project/actual/report.md", "report.md", true},
		{`C:\Project\Actual`, `c:\project\actual\report.md`, "report.md", true},
		{"/project/actual", "/project/actual/../escape.md", "", false},
		{"/project/actual", "/project/actual-other/report.md", "", false},
	} {
		rel, err := TargetRelative(projectflow.FileTarget{
			ProjectID:            "project",
			CoordinatorSessionID: "coordinator",
			Root:                 tc.root,
			CanonicalPath:        tc.p,
		}, lexicalTestValidator)
		if (err == nil) != tc.valid || rel != tc.want {
			t.Fatalf("%s: got (%q, %v), want (%q, valid=%v)", tc.p, rel, err, tc.want, tc.valid)
		}
	}
}

func TestProjectFileLifetimeDoesNotFollowChatSelection(t *testing.T) {
	root := t.TempDir()
	store, client := sessionStoreFixture(t, []map[string]any{
		{"sessionId": "composer-owner", "cwd": root},
		{"sessionId": "another-chat", "cwd": root},
	}, "composer-owner")

	ep := harness.Endpoint{Origin: client.Endpoint.Origin, Cookie: "fixture=local"}
	snap := HostSnapshot{
		Client: client,
		Status: harness.Status{State: harness.Ready, Generation: 4, Endpoint: ep},
	}
	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot { return snap },
		Sessions:     func() *session.Store { return store },
	})

	if !mgr.FileAccessCurrent(store, client, snap.Status) {
		t.Fatal("initial Project access rejected")
	}
	store.Select("another-chat")
	if !mgr.FileAccessCurrent(store, client, snap.Status) {
		t.Fatal("switching chat revoked retained Project file access")
	}
	snap.Status.Generation++
	if mgr.FileAccessCurrent(store, client, harness.Status{State: harness.Ready, Generation: 4, Endpoint: ep}) {
		t.Fatal("replaced Host retained Project file access")
	}
}

func TestPRViewKeepsExistingOfflineRootScope(t *testing.T) {
	root := t.TempDir()
	store, _ := sessionStoreFixture(t, []map[string]any{
		{"sessionId": "offline-chat", "cwd": root},
		{"sessionId": "replacement-chat", "cwd": root},
	}, "offline-chat")
	store.Detach()

	snap := HostSnapshot{
		Client: nil,
		Status: harness.Status{State: harness.Stopped},
	}
	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot { return snap },
		Sessions:     func() *session.Store { return store },
	})

	current := mgr.CapturePRViewOwner(root)
	if !current() {
		t.Fatal("existing local Git view acquired a new Host prerequisite")
	}
	store.Select("replacement-chat")
	if current() {
		t.Fatal("old PR view owner survived workspace navigation")
	}
}

func TestBindNativeConsumersSettingsAndAuthorizedFile(t *testing.T) {
	root := t.TempDir()
	store, client := sessionStoreFixture(t, []map[string]any{
		{"sessionId": "lead-1", "cwd": root},
	}, "lead-1")

	ep := harness.Endpoint{Origin: client.Endpoint.Origin, Cookie: "cookie=1"}
	hostStatus := harness.Status{State: harness.Ready, Generation: 9, Endpoint: ep}

	var settingsOpened atomic.Bool
	var openedReq *AuthorizedFileRequest
	var invalidated atomic.Int32
	mgr := NewManager(HostDeps{
		HostSnapshot:         func() HostSnapshot { return HostSnapshot{Client: client, Status: hostStatus} },
		Sessions:             func() *session.Store { return store },
		ValidateRelativeFile: lexicalTestValidator,
		ManagedMarketProfile: func(st harness.Status) string { return "local-profile" },
		OpenModelSettings:    func() { settingsOpened.Store(true) },
		OpenAuthorizedFile: func(_ any, req AuthorizedFileRequest) error {
			openedReq = &req
			return nil
		},
		Invalidate: func(_ any) { invalidated.Add(1) },
	})

	m := mgr.BindModel("win", "system", true)
	mgr.BindNativeConsumers("win", &m)

	m.OnOpenModelSettings()
	if !settingsOpened.Load() {
		t.Fatal("OnOpenModelSettings did not delegate to HostDeps")
	}

	target := projectflow.FileTarget{
		ProjectID:            "p1",
		CoordinatorSessionID: "lead-1",
		Root:                 root,
		CanonicalPath:        filepath.Join(root, "doc.md"),
	}
	m.OnOpenAuthorizedFile(target)
	if openedReq == nil || !openedReq.Local || openedReq.HostGeneration != 9 || openedReq.Target.ProjectID != "p1" {
		t.Fatalf("OpenAuthorizedFile received unexpected request: %+v", openedReq)
	}
	if err := openedReq.CheckPath(filepath.Join(root, "../escape.md")); err == nil {
		t.Fatal("CheckPath admitted traversal path")
	}
	if invalidated.Load() == 0 {
		t.Fatal("expected Invalidate after opening authorized file")
	}
}

func TestMountPolicyMatchesInstalledGates(t *testing.T) {
	for _, item := range []struct {
		cfg  config.Config
		env  string
		want bool
	}{
		{config.Defaults(), "", false},
		{config.Config{ProjectEnabled: true}, "", true},
		{config.Config{}, "1", true},
		{config.Config{}, "true", false},
		{config.Config{ProjectEnabled: true, SkipUserPlugins: true}, "1", false},
		{config.Config{ProjectEnabled: true, DisabledPlugins: []string{"dsh-project"}}, "1", false},
	} {
		if got := MountEnabled(item.cfg, item.env); got != item.want {
			t.Fatalf("MountEnabled(%+v, %q)=%v want=%v", item.cfg, item.env, got, item.want)
		}
	}
}

func TestPreferenceAccessAndSetCancellation(t *testing.T) {
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = cfg.Update(func(c *config.Config) { c.ProjectEnabled = true })

	notified := make(chan struct{}, 8)
	var restarts atomic.Int32
	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot {
			return HostSnapshot{HasSupervisor: true, SupervisorID: "sup-1"}
		},
		Config:     func() *config.Store { return cfg },
		MainWindow: func() any { return "win" },
		UpdateWindow: func(_ any, fn func()) {
			if fn != nil {
				fn()
			}
			select {
			case notified <- struct{}{}:
			default:
			}
		},
		ProtectTaskAction: func(_ context.Context, _ any, _ string, _ func() error) (bool, error) {
			return false, taskcontrol.ErrCancelled
		},
		RestartHost: func(context.Context) error {
			restarts.Add(1)
			return nil
		},
	})

	access := mgr.PreferenceAccess()
	if access == nil || !access.State.Enabled || access.OnSet == nil {
		t.Fatalf("unexpected PreferenceAccess: %+v", access)
	}
	access.OnSet(false)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snap := mgr.PreferenceAccess()
		if !snap.State.Busy && snap.State.RestartCancelled {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	snap := mgr.PreferenceAccess()
	if !snap.State.Enabled || !snap.State.RestartCancelled || restarts.Load() != 0 {
		t.Fatalf("expected cancelled restart to preserve Enabled=true and RestartCancelled=true, got %+v", snap.State)
	}
}

func TestWorkspaceDirectoryActionsAndDirectoryRefusal(t *testing.T) {
	root := t.TempDir()
	refusedDir := filepath.Join(root, "refused")
	activeDir := filepath.Join(root, "active")

	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = cfg.Update(func(c *config.Config) { c.ProjectEnabled = true })

	client := &dsh.Client{}
	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} },
		Config:       func() *config.Store { return cfg },
	})
	mgr.mu.Lock()
	mgr.primary.flow.client = client
	mgr.primary.flow.loaded = true
	mgr.primary.flow.listing = projectflow.Listing{
		Available: true,
		Projects: []projectflow.Project{
			{ID: "p1", Lifecycle: "ready", CanonicalWorkingDirectory: activeDir},
		},
	}
	mgr.primary.flow.directoryChecks = map[string]*directoryCheck{
		directoryKey(refusedDir): {reason: "unsupported directory"},
	}
	mgr.mu.Unlock()

	actions := mgr.DirectoryActions("win")
	if len(actions) != 1 {
		t.Fatalf("expected 1 directory action, got %d", len(actions))
	}
	if !actions[0].Visible(activeDir) || actions[0].Visible(refusedDir) {
		t.Fatalf("visibility mismatch: active=%v refused=%v", actions[0].Visible(activeDir), actions[0].Visible(refusedDir))
	}
	if label := actions[0].LabelForPath(activeDir); label == "" {
		t.Fatal("expected non-empty label for active directory")
	}
}

func TestCreateWithDraftCarryAndOccupiedLead(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/api/session/list":
			value = map[string]any{"items": []any{map[string]any{"sessionId": "src-1", "cwd": dir}}}
		case "/dsh-project/create":
			value = map[string]any{"project": map[string]any{"id": "p1", "coordinatorSessionId": "lead-new", "lifecycle": "ready"}}
		case "/dsh-project/detail":
			value = map[string]any{"project": map[string]any{"id": "p1", "coordinatorSessionId": "lead-new", "lifecycle": "ready"}}
		case "/dsh-project/list":
			value = map[string]any{"available": true, "projects": []any{map[string]any{"id": "p1", "coordinatorSessionId": "lead-new", "lifecycle": "ready"}}}
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
	defer srv.Close()

	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = cfg.Update(func(c *config.Config) { c.Workspace = dir })

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: srv.Client()}
	store := session.New()
	store.Attach(client, nil)
	t.Cleanup(store.Detach)
	store.RefreshList(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(store.Snapshot().Items) == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	store.Select("src-1")

	applied := make(chan string, 1)
	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} },
		Sessions:     func() *session.Store { return store },
		Config:       func() *config.Store { return cfg },
		UpdateWindow: func(_ any, fn func()) {
			if fn != nil {
				fn()
			}
		},
		CaptureWorkspaceAdoption: func(_ *session.Store) any { return "adopted-token" },
		ApplyWorkspaceAdoption: func(adoption any, _ *session.Store, sourceID, leadID string) bool {
			applied <- sourceID + "->" + leadID
			return false // Simulate Lead already occupied so error must surface cleanly.
		},
	})
	mgr.mu.Lock()
	mgr.primary.flow.client = client
	mgr.primary.flow.loaded = true
	mgr.primary.flow.loadAttempted = true
	mgr.primary.flow.listing = projectflow.Listing{Available: true}
	mgr.mu.Unlock()

	mgr.CreateWithDraft("win", dir)

	select {
	case route := <-applied:
		if route != "src-1->lead-new" {
			t.Fatalf("unexpected adoption route %q", route)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ApplyWorkspaceAdoption")
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mgr.mu.Lock()
		createErr := mgr.primary.flow.createError
		busy := mgr.primary.flow.busy
		mgr.mu.Unlock()
		if !busy && createErr != "" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected createError when ApplyWorkspaceAdoption reports Lead occupied")
}
