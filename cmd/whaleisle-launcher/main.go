// whaleisle-launcher is a standalone native surface. It never starts a Host in
// its own process or bundles the desktop runtime it manages.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/externallauncher"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/launcher"
)

var buildVersion = "dev"

type state struct {
	cancel                          context.CancelFunc
	mu                              sync.Mutex
	win                             *mygo.Window
	controller                      *externallauncher.Controller
	installed                       externallauncher.Installation
	running, busy                   bool
	message, problem, tab, selected string
	relBusy                         bool
	relStatus, relErr, relPhase     string
	relSettled                      string
	rows                            []update.NativeRelease
	staged                          update.ApplyResult
	stagedVersion                   string
	installAsked                    bool
	received, total                 int64
	// Release download control ported from the desktop update center.
	route           string // mutex-guarded; the route network calls use
	routeUI         string // view-goroutine owned; Model.DownloadRoute points here
	autoDownload    bool
	paused          bool
	pauseRequested  bool
	cancelRequested bool
	active          update.NativeRelease
	requireRuntime  bool
	staging         string
	routeUnavailable map[string]string
}

// launcherPrefs is the standalone launcher's small persisted preference file.
// It is deliberately separate from the desktop's config store.
type launcherPrefs struct {
	DownloadRoute string `json:"downloadRoute"`
	AutoDownload  bool   `json:"autoDownload"`
}

func prefsPath() string {
	return filepath.Join(paths.UserDataDir(), "launcher-prefs.json")
}

func loadPrefs() launcherPrefs {
	var p launcherPrefs
	data, err := os.ReadFile(prefsPath())
	if err == nil {
		_ = json.Unmarshal(data, &p)
	}
	return p
}

func (s *state) savePrefs() {
	s.mu.Lock()
	p := launcherPrefs{DownloadRoute: s.route, AutoDownload: s.autoDownload}
	s.mu.Unlock()
	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	if err := os.WriteFile(prefsPath(), data, 0600); err != nil {
		log.Print(err)
	}
}

func main() {
	if runtime.GOOS != "windows" {
		log.Print("The standalone native launcher currently requires Windows.")
		return
	}
	mygo.App.SetName("鲸屿 MyGo Launcher")
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	prefs := loadPrefs()
	s := &state{controller: externallauncher.Native(), tab: launcher.TabHome, route: prefs.DownloadRoute, routeUI: prefs.DownloadRoute, autoDownload: prefs.AutoDownload, routeUnavailable: map[string]string{}}
	if _, err := update.NativeCatalogURL("cnb"); err != nil {
		s.routeUnavailable["cnb"] = "CNB has no configured Whale Isle MyGo release source yet — use GitHub."
	}
	mygo.App.OnSecondInstance(func([]string, string) {
		if s.win != nil {
			s.win.Show()
			s.win.Focus()
		}
	})
	mygo.App.WhenReady(func() {
		s.win = mygo.NewWindow(mygo.WindowOptions{Title: "鲸屿 MyGo Launcher", Width: 1040, Height: 740, Content: ui.View(s.view)})
		s.win.OnClose(func(e *mygo.CloseEvent) {
			s.mu.Lock()
			busy := s.busy
			s.mu.Unlock()
			if busy {
				e.PreventDefault()
				s.notice("Wait for the current operation to finish before closing.")
			}
		})
		s.win.OnClosed(func() { mygo.App.Quit() })
		s.refresh()
		// Fetch the catalog once at startup so the auto-download preference and
		// the update badge work before the user opens the versions tab.
		s.releases()
		done := make(chan struct{})
		s.win.OnClosed(func() { close(done) })
		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					s.refresh()
				}
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Print(err)
	}
}
func (s *state) notify() {
	if s.win != nil {
		s.win.Invalidate()
	}
}
func (s *state) notice(message string)                      { s.mu.Lock(); s.message = message; s.mu.Unlock(); s.notify() }
func (s *state) operation(work func(context.Context) error) { s.operationMode(work, true, false, nil) }

// releaseOperation runs a download/stage/install operation. Only these feed
// the release strip — controller start/stop and catalog lookups stay on the
// generic busy/problem/message channels so unrelated state never swaps the
// install card's action slot.
func (s *state) releaseOperation(work func(context.Context) error, cancellable bool) {
	s.operationMode(work, cancellable, true, nil)
}
func (s *state) operationMode(work func(context.Context) error, cancellable, release bool, settled func()) {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	if cancellable {
		s.cancel = cancel
	}
	s.busy = true
	s.problem = ""
	if release {
		s.relBusy = true
		s.relErr = ""
		s.relStatus = ""
		s.relPhase = ""
	}
	s.mu.Unlock()
	s.notify()
	go func() {
		defer cancel()
		err := work(ctx)
		installed, running, probeErr := s.controller.Snapshot()
		s.mu.Lock()
		s.busy = false
		s.cancel = nil
		s.installed = installed
		s.running = running
		if release {
			s.relBusy = false
			s.relPhase = ""
			s.relStatus = s.relSettled
			s.relSettled = ""
		}
		if err != nil {
			if release {
				s.relErr = err.Error()
			} else {
				s.problem = err.Error()
			}
		} else if probeErr != nil && !errors.Is(probeErr, externallauncher.ErrNotInstalled) {
			s.problem = probeErr.Error()
		}
		s.mu.Unlock()
		s.notify()
		if settled != nil {
			settled()
		}
	}()
}
func (s *state) refresh() {
	s.mu.Lock()
	busy := s.busy
	s.mu.Unlock()
	if busy {
		return
	}
	go func() {
		installed, running, err := s.controller.Snapshot()
		s.mu.Lock()
		if !s.busy {
			s.installed = installed
			s.running = running
			if err != nil && !errors.Is(err, externallauncher.ErrNotInstalled) {
				s.problem = err.Error()
			}
		}
		s.mu.Unlock()
		s.notify()
	}()
}
func (s *state) releases() {
	s.operationMode(func(ctx context.Context) error {
		s.mu.Lock()
		route := s.route
		s.mu.Unlock()
		rows, err := update.ListNativeReleases(ctx, route)
		if err != nil {
			return err
		}
		s.mu.Lock()
		version := s.installed.Version
		missing := s.installed.Executable == ""
		s.mu.Unlock()
		rows = update.PrepareNativeReleases(rows, version, update.NativeInstallerCacheDir(filepath.Join(paths.UserDataDir(), "updates")))
		// A fresh install requires a runtime-inclusive package; shell-only rows
		// carry the reason in the list instead of failing on download click.
		for i := range rows {
			if rows[i].UnavailableReason == "" {
				if verr := rows[i].Validate(runtime.GOOS, runtime.GOARCH, missing); verr != nil {
					rows[i].UnavailableReason = verr.Error()
				}
			}
		}
		s.mu.Lock()
		s.rows = rows
		s.message = "Choose a native desktop package to download and verify."
		s.mu.Unlock()
		return nil
	}, true, false, s.maybeAutoDownload)
}
func (s *state) stage(id string) {
	s.releaseOperation(func(ctx context.Context) error {
		return s.stageWork(ctx, nil, id)
	}, true)
}

// resume re-downloads the pinned active release from its retained partial
// staging directory. The staging layer binds a .part file to the release's
// URL, hash and size, so the same deterministic directory resumes byte-wise.
func (s *state) resume() {
	s.mu.Lock()
	if s.relBusy || s.busy || s.active.PackageKey() == "" {
		s.mu.Unlock()
		return
	}
	pinned := s.active
	s.mu.Unlock()
	s.releaseOperation(func(ctx context.Context) error {
		return s.stageWork(ctx, &pinned, "")
	}, true)
}

// pause cancels the in-flight download but marks pauseRequested first: the
// stage tail then keeps the partial directory and exposes the paused state
// instead of an error, matching pauseNativeUpdate in the desktop app.
func (s *state) pause() {
	s.mu.Lock()
	if s.relPhase != "downloading" || !s.relBusy {
		s.mu.Unlock()
		return
	}
	s.pauseRequested = true
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.notify()
}

// cancelRelease stops a running download and discards its partial data, or
// drops the paused state (and its retained partial) when nothing is running.
func (s *state) cancelRelease() {
	s.mu.Lock()
	if s.relBusy {
		s.cancelRequested = true
		s.pauseRequested = false
		cancel := s.cancel
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		s.notify()
		return
	}
	if !s.paused && s.staged.Path == "" {
		s.mu.Unlock()
		return
	}
	dir := s.staging
	s.paused = false
	s.active = update.NativeRelease{}
	s.staging = ""
	s.staged = update.ApplyResult{}
	s.stagedVersion = ""
	s.received, s.total = 0, 0
	s.relErr = ""
	s.relStatus = "Download cancelled."
	s.mu.Unlock()
	if dir != "" {
		_ = os.RemoveAll(dir)
	}
	s.notify()
}

func (s *state) setAutoDownload(enabled bool) {
	s.mu.Lock()
	s.autoDownload = enabled
	s.mu.Unlock()
	s.savePrefs()
	if enabled {
		s.maybeAutoDownload()
	}
	s.notify()
}

// maybeAutoDownload stages the newest compatible stable package when the user
// opted in. Errors, a paused download, and a ready staged installer all stop
// it — like the desktop's maybeAutoDownloadNative.
func (s *state) maybeAutoDownload() {
	s.mu.Lock()
	allowed := s.autoDownload && !s.busy && !s.relBusy && !s.paused && s.relErr == "" && s.staged.Path == ""
	missing := s.installed.Executable == ""
	current := s.installed.Version
	available := false
	if allowed {
		for _, release := range s.rows {
			if !release.Prerelease && release.UnavailableReason == "" && (!missing || release.RuntimeIncluded) && update.CompareVersions(release.Version, current) > 0 {
				available = true
				break
			}
		}
	}
	s.mu.Unlock()
	if available {
		s.stage("")
	}
}

// saveRouteUI is the Model.OnSaveLauncherPrefs hook: the route chips and popup
// write through *m.DownloadRoute (view goroutine), then this copies it under
// the mutex, persists it, and refetches the catalog on the new route.
func (s *state) saveRouteUI() {
	s.mu.Lock()
	changed := s.routeUI != "" && s.routeUI != s.route
	if s.routeUI != "" {
		s.route = s.routeUI
	}
	s.mu.Unlock()
	s.savePrefs()
	if changed {
		s.releases()
	}
}

// stagingDirFor keeps one partial directory per package so an interrupted
// download resumes byte-wise — even across launcher restarts.
func stagingDirFor(key string) string {
	var b strings.Builder
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	name := b.String()
	if name == "" {
		name = "release"
	}
	return filepath.Join(paths.UserDataDir(), "updates", "native-partials", name)
}

func (s *state) stageWork(ctx context.Context, pinned *update.NativeRelease, id string) error {
	s.mu.Lock()
	s.relStatus = "Downloading and verifying the installer…"
	s.relPhase = "downloading"
	s.paused, s.pauseRequested, s.cancelRequested = false, false, false
	route := s.route
	version := s.installed.Version
	missing := s.installed.Executable == ""
	s.mu.Unlock()
	s.notify()

	cache := update.NativeInstallerCacheDir(filepath.Join(paths.UserDataDir(), "updates"))
	var selected update.NativeRelease
	var err error
	if pinned != nil {
		selected = *pinned
		err = selected.Validate(runtime.GOOS, runtime.GOARCH, missing)
	} else {
		var rows []update.NativeRelease
		rows, err = update.ListNativeReleases(ctx, route)
		if err == nil {
			rows = update.PrepareNativeReleases(rows, version, cache)
			selected, err = update.SelectNativeRelease(rows, id, runtime.GOOS, runtime.GOARCH, missing)
			if err == nil {
				s.mu.Lock()
				s.rows = rows
				s.mu.Unlock()
			}
		}
	}
	if err != nil {
		return err
	}
	dir := stagingDirFor(selected.PackageKey())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	s.mu.Lock()
	s.active = selected
	s.requireRuntime = missing
	s.staging = dir
	s.staged, s.stagedVersion = update.ApplyResult{}, ""
	s.received, s.total = 0, 0
	s.mu.Unlock()
	s.notify()
	staged, err := update.StageNativeReleaseWithOptions(ctx, selected, update.NativeStageOptions{Resume: true, CurrentVersion: version, CacheDir: cache, RequireRuntime: missing, DestDir: dir, OnPhase: func(phase update.StagePhase) {
		s.mu.Lock()
		s.relPhase = string(phase)
		s.mu.Unlock()
		s.notify()
	}, OnProgress: func(received, total int64) {
		s.mu.Lock()
		s.received = received
		s.total = total
		s.mu.Unlock()
		s.notify()
	}})
	if err != nil {
		s.mu.Lock()
		pauseRequested := s.pauseRequested
		s.pauseRequested, s.cancelRequested = false, false
		switch {
		case errors.Is(err, context.Canceled) && pauseRequested:
			// Keep the partial directory and the pinned release: resume picks
			// up exactly where this attempt stopped.
			s.paused = true
			s.relSettled = "Download paused — resume to continue, or cancel to discard the partial download."
			s.mu.Unlock()
			return nil
		case errors.Is(err, context.Canceled):
			s.active = update.NativeRelease{}
			s.paused = false
			s.staging = ""
			s.received, s.total = 0, 0
			s.relSettled = "Download cancelled."
			s.mu.Unlock()
			_ = os.RemoveAll(dir)
			return nil
		default:
			// A failed download stays resumable: the strip shows the error and
			// the resume button retries from the retained partial data.
			s.paused = true
			s.mu.Unlock()
			return err
		}
	}
	s.mu.Lock()
	s.staged = staged
	s.stagedVersion = selected.Version
	s.selected = selected.PackageKey()
	s.relStatus = ""
	s.message = "Installer verified. Stop the desktop first, then install. This launcher remains open."
	s.mu.Unlock()
	return nil
}
func (s *state) install() {
	s.mu.Lock()
	staged, version, busy := s.staged, s.stagedVersion, s.busy
	if busy || staged.Path == "" || s.installAsked {
		s.mu.Unlock()
		return
	}
	s.installAsked = true
	s.mu.Unlock()
	// Dialog.Message must not run inside the click handler: the handler's
	// event dispatch holds the main thread, so the posted dialog work would
	// never drain. Ask on a goroutine like startNativeStaged does.
	go func() {
		defer func() {
			s.mu.Lock()
			s.installAsked = false
			s.mu.Unlock()
		}()
		s.installConfirmed(staged, version)
	}()
}
func (s *state) installConfirmed(staged update.ApplyResult, version string) {
	choice, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: s.win, Type: mygo.MessageWarning, Title: "Install native desktop", Message: fmt.Sprintf("Install verified desktop %s? The desktop must be stopped first. The installer will ask for its destination. This launcher waits and adopts the registered installation; it does not quit.", version), Buttons: []string{"Cancel", "Install"}, DefaultButton: 0, CancelButton: 0})
	if err != nil || choice.Button != 1 {
		return
	}
	s.releaseOperation(func(ctx context.Context) error {
		s.mu.Lock()
		s.relStatus = "Installing…"
		s.relPhase = "installing"
		s.mu.Unlock()
		fresh, err := s.controller.Install(ctx, staged, version)
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.installed = fresh
		s.staged = update.ApplyResult{}
		s.stagedVersion = ""
		s.relStatus = ""
		s.message = "Installation registered and adopted. Start desktop when ready."
		s.mu.Unlock()
		return nil
	}, false)
}
func (s *state) view(c *ui.Context) {
	s.mu.Lock()
	m := launcher.Model{ThemeID: "light", Version: buildVersion, LauncherPackage: true, ManagedDesktop: true, VendorReady: s.installed.Executable != "", DesktopRunning: s.running, StartBusy: s.busy, StartError: s.problem, StatusNote: s.message, ActiveTab: s.tab, Releases: append([]update.NativeRelease(nil), s.rows...), ReleaseBusy: s.relBusy, ReleaseError: s.relErr, ReleaseStatus: s.relStatus, ReleasePhase: s.relPhase, ReleaseSelectedPackage: s.selected, ReleaseStagedTag: s.stagedVersion, ReleaseReceived: s.received, ReleaseTotal: s.total, ReleasePaused: s.paused, ReleaseAutoDownload: s.autoDownload, DownloadRoute: &s.routeUI, RouteUnavailable: s.routeUnavailable}
	if s.installed.Version != "" {
		m.Version = s.installed.Version
	}
	s.mu.Unlock()
	m.OnCancelRelease = s.cancelRelease
	m.OnPauseRelease = s.pause
	m.OnResumeRelease = s.resume
	m.OnSetAutoDownload = s.setAutoDownload
	m.OnSaveLauncherPrefs = s.saveRouteUI
	m.OnDismissRelease = func() {
		s.mu.Lock()
		s.relErr = ""
		s.relStatus = ""
		s.mu.Unlock()
		s.notify()
	}
	m.OnStart = func() { s.operation(s.controller.Start) }
	m.OnStop = func() {
		s.notice("Confirm any task-protection prompt in the desktop. No process will be force-killed.")
		s.operation(s.controller.Stop)
	}
	m.OnClose = func() { s.win.Close() }
	m.OnTab = func(tab string) {
		if tab != launcher.TabHome && tab != launcher.TabVersions {
			s.notice("Open the installed desktop to manage its settings, plugins, components and imports.")
			return
		}
		s.mu.Lock()
		s.tab = tab
		s.mu.Unlock()
		s.notify()
		if tab == launcher.TabVersions {
			s.releases()
		}
	}
	m.OnCheckUpdate = s.releases
	m.OnRefreshReleases = s.releases
	m.OnInstallRuntime = func() { s.stage("") }
	m.OnInstallUpdate = m.OnInstallRuntime
	m.OnInstallRelease = s.stage
	m.OnStartStagedRelease = s.install
	m.OnSelectRelease = func(id string) {
		s.mu.Lock()
		if !s.busy {
			s.selected = id
		}
		s.mu.Unlock()
		s.notify()
	}
	launcher.View(c, m)
}
