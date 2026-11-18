package projectctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/projects"
)

func TestObserverControllersRetainIndependentState(t *testing.T) {
	client := &dsh.Client{}
	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} },
	})
	mgr.mu.Lock()
	mgr.primary.flow.client = client
	mgr.mu.Unlock()

	a := &projects.ObserverState{ProjectID: "a", CoordinatorSessionID: "lead-a", Section: "tasks"}
	b := &projects.ObserverState{ProjectID: "b", CoordinatorSessionID: "lead-b", Section: "materials"}
	ca := mgr.Observer(a)
	cb := mgr.Observer(b)
	if ca == cb || ca == mgr.Primary() || cb == mgr.Primary() {
		t.Fatal("observers share the mutable global controller")
	}

	mgr.mu.Lock()
	ca.flow.requestProjectID = "a"
	ca.flow.requestKind = "completed"
	ca.flow.requestCursors = []string{"", "a-page-2"}
	ca.flow.requestCard = &projectflow.RequestCard{ID: "a-card", ProjectID: "a"}
	cb.flow.requestProjectID = "b"
	cb.flow.requestKind = "archived"
	cb.flow.requestCursors = []string{"", "b-page-2", "b-page-3"}
	cb.flow.requestCard = &projectflow.RequestCard{ID: "b-card", ProjectID: "b"}
	ca.flow.materials = map[string]*projectflow.MaterialDraft{"a": {ProjectID: "a", Draft: "unsaved a", Saved: "a", Loaded: true}}
	cb.flow.materials = map[string]*projectflow.MaterialDraft{"b": {ProjectID: "b", Draft: "unsaved b", Saved: "b", Loaded: true}}
	mgr.mu.Unlock()

	ma := ca.Model(nil)
	_ = cb.Model(nil)
	ma.OnBackRequests()

	mgr.mu.Lock()
	if ca.flow.requestCard != nil || cb.flow.requestCard == nil || cb.flow.requestCard.ID != "b-card" {
		mgr.mu.Unlock()
		t.Fatal("A callback modified B's selected card")
	}
	if len(ca.flow.requestCursors) != 2 || len(cb.flow.requestCursors) != 3 || cb.flow.requestKind != "archived" {
		mgr.mu.Unlock()
		t.Fatal("request pagination did not remain independently retained")
	}
	if !ca.flow.materials["a"].Dirty() || !cb.flow.materials["b"].Dirty() {
		mgr.mu.Unlock()
		t.Fatal("switching observer lost material drafts")
	}
	if mgr.primary.flow.requestCard != nil || mgr.primary.flow.requestProjectID != "" {
		mgr.mu.Unlock()
		t.Fatal("observer mutated sidebar controller")
	}
	mgr.mu.Unlock()

	if mgr.Observer(a) != ca || mgr.Observer(b) != cb {
		t.Fatal("switching coordinator replaced its observer controller")
	}

	ca.Dispose()
	mgr.mu.Lock()
	bClient := cb.flow.client
	bCard := cb.flow.requestCard
	mgr.mu.Unlock()
	if bClient != client || bCard == nil {
		t.Fatal("disposing A invalidated B")
	}
}

func TestOrdinaryTeamModelInitialLoadingAndPreservation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RPCID   string
			Payload map[string]any
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch r.URL.Path {
		case "/api/session/list":
			value = map[string]any{"items": []any{map[string]any{"sessionId": "s1", "running": false}}}
		case "/api/session/projections":
			value = map[string]any{
				"values": map[string]any{
					"agentTeam": map[string]any{
						"members": []any{map[string]any{"id": "s1", "name": "lead", "role": "lead", "phase": "active"}},
						"tasks":   []any{},
					},
				},
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
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: srv.Client()}
	store := session.New()
	store.Select("s1")

	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} },
		Sessions:     func() *session.Store { return store },
	})

	// First call must immediately reflect Loading=true (no 1-frame flash of unavailable).
	m := mgr.OrdinaryTeamModel(nil)
	if !m.Loading && m.Team == nil {
		t.Fatal("first frame of OrdinaryTeamModel returned Loading=false && Team==nil")
	}

	// BindModel when primary.flow.client transitions to client must preserve ordinaryTeam.
	_ = mgr.BindModel(nil, "system", false)
	mgr.mu.Lock()
	preserved := mgr.ordinaryTeam != nil
	mgr.mu.Unlock()
	if !preserved {
		t.Fatal("BindModel wiped out ordinaryTeam")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m = mgr.OrdinaryTeamModel(nil)
		if m.Team != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if m.Team == nil || len(m.Team.Members) != 1 || m.Team.Members[0].Name != "lead" {
		t.Fatalf("expected loaded Team with member 'lead', got %+v (err=%q)", m.Team, m.Error)
	}
}

func TestModelsKeepTheirFrameTheme(t *testing.T) {
	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			client := &dsh.Client{}
			mgr := NewManager(HostDeps{
				HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: client} },
			})
			mgr.mu.Lock()
			mgr.primary.flow.client = client
			mgr.mu.Unlock()

			themeID := "frame-theme"
			project := mgr.BindModel(nil, themeID, dark)
			if project.ThemeID != "frame-theme" || project.SystemDark != dark || project.VisualPage == nil {
				t.Fatal("Project did not bind with the frame theme")
			}

			// Mutating caller variables after BindModel must not alter the captured observer closure.
			themeID = "next-frame"
			observer := project.ObserverModel(&projects.ObserverState{Section: "tasks"})
			_ = themeID
			if observer.ThemeID != "frame-theme" || observer.SystemDark != dark || observer.VisualPage == nil {
				t.Fatal("observer lost its captured frame theme")
			}
		})
	}
}

func TestLockDecouplingNeverHoldsManagerMutexInHostDeps(t *testing.T) {
	var hostMu sync.Mutex
	client := &dsh.Client{}
	store := session.New()
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = cfg.Update(func(c *config.Config) { c.ProjectEnabled = true })

	var mgr *Manager
	var singleThreadCheck atomic.Bool
	singleThreadCheck.Store(true)
	// A host dependency must never run with Manager.mu held. A bare TryLock
	// cannot decide that: it also fails while another goroutine holds the mutex,
	// which the refresh loop does while a model call runs, and then the failure
	// names whichever dependency happened to coincide. hostDepUnderLockOnly
	// attributes the hold to the calling goroutine before anything is reported.
	violations := make(chan string, 1)
	assertUnlocked := func(name string) {
		if mgr == nil || !singleThreadCheck.Load() {
			return
		}
		if !hostDepUnderLockOnly(mgr) {
			return
		}
		// The dependency runs on whichever goroutine called it, so record the
		// violation here and fail from the test goroutine.
		select {
		case violations <- name:
		default:
		}
	}

	mgr = NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot {
			assertUnlocked("HostSnapshot")
			hostMu.Lock()
			defer hostMu.Unlock()
			return HostSnapshot{
				Client: client,
				Status: harness.Status{State: harness.Ready, Generation: 1},
			}
		},
		LifecycleContext: func() context.Context {
			assertUnlocked("LifecycleContext")
			return context.Background()
		},
		Sessions: func() *session.Store {
			assertUnlocked("Sessions")
			hostMu.Lock()
			defer hostMu.Unlock()
			return store
		},
		Config: func() *config.Store {
			assertUnlocked("Config")
			hostMu.Lock()
			defer hostMu.Unlock()
			return cfg
		},
		UpdateWindow: func(_ any, fn func()) {
			assertUnlocked("UpdateWindow")
			if fn != nil {
				fn()
			}
		},
		Invalidate: func(_ any) {
			assertUnlocked("Invalidate")
		},
	})

	mgr.mu.Lock()
	mgr.primary.flow.client = client
	mgr.primary.flow.loaded = true
	mgr.primary.flow.loadAttempted = true
	mgr.primary.flow.listing = projectflow.Listing{
		Available: true,
		Projects:  []projectflow.Project{{ID: "p1", CoordinatorSessionID: "lead-1", Lifecycle: "ready"}},
	}
	mgr.mu.Unlock()

	// Phase 1: Single-goroutine deterministic verification that Manager.mu is never held during HostDeps.
	mSingle := mgr.BindModel("win", "system", false)
	mgr.BindNativeConsumers("win", &mSingle)
	mSingle.ObserverModel(&projects.ObserverState{ProjectID: "p1", CoordinatorSessionID: "lead-1", Section: "tasks"})
	_ = mgr.OrdinaryTeamModel("win")
	_ = mgr.TeamModel("win", client, "p1")
	_ = mgr.DirectoryActions("win")
	select {
	case name := <-violations:
		t.Fatalf("%s invoked while Manager.mu was held", name)
	default:
	}

	// Phase 2: Multi-goroutine stress verifying hostMu -> Manager.mu DAG ordering has zero deadlocks or races.
	singleThreadCheck.Store(false)
	var wg sync.WaitGroup
	var stop atomic.Bool
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				// Simulate host locking hostMu BEFORE calling into Manager (the allowed DAG order).
				hostMu.Lock()
				_, ok := mgr.LoadedProjects(client)
				hostMu.Unlock()
				if !ok {
					t.Error("expected LoadedProjects to succeed")
					return
				}
				m := mgr.BindModel(nil, "system", false)
				m.ObserverModel(&projects.ObserverState{ProjectID: "p1", CoordinatorSessionID: "lead-1", Section: "tasks"})
				_ = mgr.OrdinaryTeamModel(nil)
				_ = mgr.TeamModel(nil, client, "p1")
				_ = mgr.DirectoryActions(nil)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 40 {
			mgr.SetWindowHidden(i%2 == 0)
			mgr.Suspend()
			time.Sleep(time.Millisecond)
		}
		stop.Store(true)
	}()
	wg.Wait()
	mgr.Close()
}

func TestLoadedProjectsAndSuspendLifecycle(t *testing.T) {
	c1 := &dsh.Client{}
	c2 := &dsh.Client{}
	mgr := NewManager(HostDeps{
		HostSnapshot: func() HostSnapshot { return HostSnapshot{Client: c1} },
	})
	mgr.mu.Lock()
	mgr.primary.flow.client = c1
	mgr.primary.flow.loaded = true
	mgr.primary.flow.listing = projectflow.Listing{
		Available: true,
		Projects:  []projectflow.Project{{ID: "p1", Title: "Project 1"}},
	}
	ctx, cancel := context.WithCancel(context.Background())
	mgr.primary.flow.processCancel = cancel
	mgr.primary.flow.refreshLoop = &refreshLoop{cancel: cancel, client: c1}
	mgr.mu.Unlock()

	projectsList, ok := mgr.LoadedProjects(c1)
	if !ok || len(projectsList) != 1 || projectsList[0].ID != "p1" {
		t.Fatalf("unexpected LoadedProjects: ok=%v list=%+v", ok, projectsList)
	}
	// Returned slice must be defensive copy.
	projectsList[0].ID = "mutated"
	checkAgain, _ := mgr.LoadedProjects(c1)
	if checkAgain[0].ID != "p1" {
		t.Fatal("mutating returned slice affected Manager listing")
	}
	if _, ok := mgr.LoadedProjects(c2); ok {
		t.Fatal("LoadedProjects returned data for mismatched client")
	}

	mgr.Suspend()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("Suspend did not cancel active process/refresh context")
	}
}

// foreignHoldGrace bounds how long another goroutine may keep Manager.mu before
// the hold is attributed to the caller. Every critical section in this package
// publishes state and leaves the host dependencies outside the lock, so a
// foreign hold lasts microseconds; a lock the caller holds itself cannot clear
// while the caller is the one waiting on it.
const foreignHoldGrace = 250 * time.Millisecond

// hostDepUnderLockOnly reports whether the calling goroutine holds m.mu.
// TryLock alone cannot answer that: it fails for every holder, so a background
// refresh would be blamed on the model call that happened to run beside it.
// Waiting out a foreign critical section separates the two cases: only a lock
// this goroutine took stays taken for the whole grace period.
func hostDepUnderLockOnly(m *Manager) bool {
	if m.mu.TryLock() {
		m.mu.Unlock()
		return false
	}
	deadline := time.Now().Add(foreignHoldGrace)
	for time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
		if m.mu.TryLock() {
			m.mu.Unlock()
			return false
		}
	}
	return true
}

func TestHostDepUnderLockOnlySeesSelfHeldLock(t *testing.T) {
	m := NewManager(HostDeps{})
	m.mu.Lock()
	defer m.mu.Unlock()
	if !hostDepUnderLockOnly(m) {
		t.Fatal("guard misses a lock the calling goroutine holds")
	}
}

func TestHostDepUnderLockOnlyIgnoresForeignHolder(t *testing.T) {
	m := NewManager(HostDeps{})
	held := make(chan struct{})
	go func() {
		m.mu.Lock()
		close(held)
		time.Sleep(20 * time.Millisecond)
		m.mu.Unlock()
	}()
	<-held
	if hostDepUnderLockOnly(m) {
		t.Fatal("guard blamed the caller for another goroutine's hold")
	}
}
