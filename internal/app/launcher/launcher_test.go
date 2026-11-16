package launcherctl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/app/updater"
	"whaleisle/internal/config"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/product"
	"whaleisle/internal/store/session"
)

type fakePathSetter struct {
	name     string
	userData string
	cache    string
	logs     string
}

func (f *fakePathSetter) SetName(n string)         { f.name = n }
func (f *fakePathSetter) SetUserDataPath(p string) { f.userData = p }
func (f *fakePathSetter) SetCachePath(p string)    { f.cache = p }
func (f *fakePathSetter) SetLogsPath(p string)     { f.logs = p }

type fakeLauncherStage struct {
	haveLauncher    bool
	mainShown       bool
	mainVisible     bool
	quitting        bool
	windowAvailable bool
	created         int
	closed          int
	shows           int
	events          []string
}

func (f *fakeLauncherStage) LauncherOpen() bool { return f.haveLauncher }
func (f *fakeLauncherStage) ShowLauncher() bool {
	if !f.windowAvailable {
		return false
	}
	if !f.haveLauncher {
		f.haveLauncher = true
		f.created++
	}
	f.shows++
	f.events = append(f.events, "show-launcher")
	return true
}
func (f *fakeLauncherStage) CloseLauncher() {
	f.haveLauncher = false
	f.closed++
	f.events = append(f.events, "close-launcher")
}
func (f *fakeLauncherStage) ShowMain() {
	f.mainShown = true
	f.mainVisible = true
	f.events = append(f.events, "show-main")
}
func (f *fakeLauncherStage) MainVisible() bool { return f.mainVisible }
func (f *fakeLauncherStage) Quitting() bool    { return f.quitting }

type stubSelectionDeps struct {
	envMu sync.Mutex
	env   HostEnv
}

func (s *stubSelectionDeps) Env() HostEnv {
	s.envMu.Lock()
	defer s.envMu.Unlock()
	return s.env
}
func (s *stubSelectionDeps) LifecycleContext() context.Context                       { return context.Background() }
func (s *stubSelectionDeps) Updater() *updater.Center                                { return nil }
func (s *stubSelectionDeps) MarketBusy() bool                                        { return false }
func (s *stubSelectionDeps) TryAcquireRepairSlot() (func(), bool)                    { return func() {}, true }
func (s *stubSelectionDeps) TrySetTaskRestartPending(bool) bool                      { return true }
func (s *stubSelectionDeps) ConfigureGenericComponents()                             {}
func (s *stubSelectionDeps) SyncHarnessFromConfig()                                  {}
func (s *stubSelectionDeps) StartHarnessWithConfig(config.Config)                    {}
func (s *stubSelectionDeps) RestartSupervisorIfPresent()                             {}
func (s *stubSelectionDeps) StopSupervisorIfPresent()                                {}
func (s *stubSelectionDeps) HasCurrentSupervisor() bool                              { return false }
func (s *stubSelectionDeps) WithTaskProtection(WindowRef, string, func() error) bool { return false }
func (s *stubSelectionDeps) CloseTaskProtectedResources() error                      { return nil }
func (s *stubSelectionDeps) OpenStartupBlankSession(WindowRef, *session.Store, uint64, string) {
}
func (s *stubSelectionDeps) ObserveBridgeAlerts()           {}
func (s *stubSelectionDeps) RefuseHostAction()              {}
func (s *stubSelectionDeps) UpdateWindow(WindowRef, func()) {}
func (s *stubSelectionDeps) InvalidateSurface(WindowRef)    {}

func TestFirstWindowGatesColdStart(t *testing.T) {
	for _, tc := range []struct {
		name                                   string
		isolated, hidden, background, settings bool
		want                                   bool
	}{
		{"cold foreground start", false, false, false, false, true},
		{"isolated desktop keeps the panel", true, false, false, false, false},
		{"login start keeps the main window", false, true, false, false, false},
		{"background never creates it", false, false, true, false, false},
		{"settings route needs the main window", false, false, false, true, false},
	} {
		if got := FirstWindow(tc.isolated, tc.hidden, tc.background, tc.settings); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestLauncherStageLifecycle(t *testing.T) {
	prev := SetPortableDesktopRoot("")
	t.Cleanup(func() { SetPortableDesktopRoot(prev) })
	t.Setenv("WHALEISLE_ISOLATED", "")

	stage := &fakeLauncherStage{windowAvailable: true}
	res := ApplyStage(stage, true)
	if !res.LauncherShown || res.MainShown || res.LauncherClosed {
		t.Fatalf("cold start result = %+v", res)
	}
	for i := 0; i < 3; i++ {
		again := ApplyStage(stage, true)
		if again.LauncherShown || again.MainShown || again.LauncherClosed {
			t.Fatalf("repeat %d created work: %+v", i, again)
		}
	}
	res2 := ApplyStage(stage, false)
	if !res2.LauncherClosed || !res2.MainShown {
		t.Fatalf("hand-off = %+v", res2)
	}
	if again := ApplyStage(stage, false); again.LauncherClosed || again.MainShown {
		t.Fatalf("second hand-off = %+v", again)
	}

	fallback := &fakeLauncherStage{windowAvailable: false}
	res3 := ApplyStage(fallback, true)
	if !res3.MainShown || res3.LauncherShown || fallback.created != 0 {
		t.Fatalf("fallback = %+v", res3)
	}

	quitting := &fakeLauncherStage{windowAvailable: true, quitting: true}
	if r := ApplyStage(quitting, true); r.MainShown {
		t.Fatalf("quit showed main window: %+v", r)
	}
	if r := ApplyStage(quitting, false); !r.LauncherClosed || r.MainShown {
		t.Fatalf("quit close = %+v", r)
	}
}

func TestIsolatedDesktopGuardsAndVisibilityCombinations(t *testing.T) {
	prev := SetPortableDesktopRoot("")
	t.Cleanup(func() { SetPortableDesktopRoot(prev) })

	t.Setenv("WHALEISLE_ISOLATED", "1")
	calls := 0
	if err := DesktopSystemIntegration(func() error { calls++; return errors.New("sentinel") }); err != nil || calls != 0 {
		t.Fatalf("isolated run reached integration: calls=%d err=%v", calls, err)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{"", "relative", filepath.Join(base, config.DataDirName)} {
		t.Setenv(paths.EnvUserData, data)
		if err := PrepareIsolatedDesktop(&fakePathSetter{}); err == nil {
			t.Fatalf("accepted non-isolated data dir: %q", data)
		}
	}

	data := t.TempDir()
	t.Setenv(paths.EnvUserData, data)
	setter := &fakePathSetter{}
	if err := PrepareIsolatedDesktop(setter); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(setter.name, product.Name+" Isolated ") || setter.userData != filepath.Join(data, "native") {
		t.Fatalf("unexpected isolated setter outputs: %+v", setter)
	}
	if roots := DesktopComponentRoots(); !reflect.DeepEqual(roots, []string{data}) {
		t.Fatalf("roots = %#v", roots)
	}
	for _, tc := range []struct {
		isolated, visible        string
		wantIsolated, wantHidden bool
	}{
		{"", "", false, false},
		{"", "1", false, false},
		{"1", "", true, true},
		{"1", "1", true, false},
		{"1", "0", true, true},
	} {
		t.Setenv("WHALEISLE_ISOLATED", tc.isolated)
		t.Setenv("WHALEISLE_ISOLATED_VISIBLE", tc.visible)
		if IsolatedDesktopRun() != tc.wantIsolated || IsolatedDesktopHidden() != tc.wantHidden {
			t.Fatalf("isolated=%q visible=%q got %v/%v", tc.isolated, tc.visible, IsolatedDesktopRun(), IsolatedDesktopHidden())
		}
	}
}

func TestPortableManifestValidationAndContainment(t *testing.T) {
	prev := SetPortableDesktopRoot("")
	t.Cleanup(func() { SetPortableDesktopRoot(prev) })
	t.Setenv(paths.EnvUserData, "")
	t.Setenv(paths.EnvVendor, "")
	t.Setenv("WHALEISLE_ISOLATED", "")

	root := t.TempDir()
	if err := PreparePortableDesktopAt(filepath.Join(root, "whaleisle.exe")); err != nil {
		t.Fatal(err)
	}
	if IsolatedDesktopRun() {
		t.Fatal("missing manifest activated isolation")
	}

	private := filepath.Join(root, "data")
	if err := PortableContainedPath(private, filepath.Join(private, "config.json"), true); err != nil {
		t.Fatalf("fresh child rejected: %v", err)
	}
	if err := PortableContainedPath(private, root, false); err == nil {
		t.Fatal("outside path accepted")
	}

	raw, _ := json.Marshal(PortableDesktopManifest{2, "installed", "../x", "runtime", "WhaleIsle.exe"})
	if err := os.WriteFile(filepath.Join(root, "portable-package.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := PreparePortableDesktopAt(filepath.Join(root, "WhaleIsle.exe")); err == nil {
		t.Fatal("invalid schema accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) || PortableDesktopRoot() != "" {
		t.Fatal("invalid manifest initialized data")
	}
}

func TestLauncherUpdateProbeIsNonBlockingAndDeduplicates(t *testing.T) {
	var requests atomic.Int32
	requested := make(chan struct{}, 4)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		requested <- struct{}{}
		<-release
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(desktopupdate.Feed{Version: "1.0.0", URL: "https://example.invalid/desktop.zip"})
	}))
	defer srv.Close()
	defer unblock()
	t.Setenv(desktopupdate.EnvFeedURL, srv.URL)

	c := NewController(nil)
	c.SetUpdateAvailableForTest(true)
	c.CheckUpdate(nil, "1.0.0")
	c.CheckUpdate(nil, "1.0.0")

	select {
	case <-requested:
	case <-time.After(3 * time.Second):
		t.Fatal("update check never issued HTTP request")
	}
	if !c.UpdateProbeState().CheckBusy {
		t.Fatal("expected CheckBusy while HTTP blocked")
	}
	select {
	case <-requested:
		t.Fatal("duplicate check started second HTTP call")
	case <-time.After(60 * time.Millisecond):
	}
	unblock()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		st := c.UpdateProbeState()
		if !st.CheckBusy {
			if st.Available || !strings.Contains(st.Message, string(desktopupdate.StatusUpToDate)) {
				t.Fatalf("unexpected probe state: %+v", st)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for probe completion")
}

func TestCommitDraftsDifferentialGuardAndStartupSelection(t *testing.T) {
	c := NewController(nil)
	cfg := config.Defaults()
	cfg.Workspace = "C:/ws1"
	snap := c.PanelState(&cfg, "")
	if snap.Drafts.Workspace != "C:/ws1" {
		t.Fatalf("seeded workspace = %q", snap.Drafts.Workspace)
	}
	before := snap.Drafts
	next := before
	next.QuitAfterStart = !before.QuitAfterStart
	// Reset workspace asynchronously, simulating Start or concurrent reset.
	c.ResetWorkspaceDraft("C:/ws2")
	next.Workspace = "C:/stale-input"
	c.CommitDrafts(&before, next)
	after := c.PanelState(&cfg, "")
	if after.Drafts.Workspace != "C:/ws2" {
		t.Fatalf("stale workspace draft overwrote newer reset: %q", after.Drafts.Workspace)
	}
	if after.Drafts.QuitAfterStart != next.QuitAfterStart {
		t.Fatal("changed quit toggle did not commit")
	}

	// Startup blank-session claim consumes once per binding and retires on panel open.
	type winKey struct{}
	w := &winKey{}
	sess := session.New()
	d := &stubSelectionDeps{env: HostEnv{
		HasSessions: true,
		MainWindow:  w,
		BootStatus:  harness.Status{Generation: 9},
	}}
	sc := NewController(d)
	receipt := session.InitialWorkspaceSession{Binding: 4, HostGeneration: 9, WorkspaceID: "default"}
	if sc.ClaimStartupSelection(w, sess, receipt) {
		t.Fatal("claimed before Ready")
	}
	receipt.Ready = true
	if !sc.ClaimStartupSelection(w, sess, receipt) {
		t.Fatal("failed to claim ready startup blank session")
	}
	if sc.ClaimStartupSelection(w, sess, receipt) {
		t.Fatal("claimed same binding twice")
	}
	receipt.Binding++
	d.envMu.Lock()
	d.env.ExclusivePanelOpen = true
	d.envMu.Unlock()
	if sc.ClaimStartupSelection(w, sess, receipt) {
		t.Fatal("claimed while exclusive panel open")
	}
}
