// whaleisle-launcher is a standalone native surface. It never starts a Host in
// its own process or bundles the desktop runtime it manages.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
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
	rows                            []update.NativeRelease
	staged                          update.ApplyResult
	stagedVersion                   string
	received, total                 int64
}

func main() {
	if runtime.GOOS != "windows" {
		log.Print("The standalone native launcher currently requires Windows.")
		return
	}
	mygo.App.SetName("MyGo鲸屿 Launcher")
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	s := &state{controller: externallauncher.Native(), tab: launcher.TabHome}
	mygo.App.OnSecondInstance(func([]string, string) {
		if s.win != nil {
			s.win.Show()
			s.win.Focus()
		}
	})
	mygo.App.WhenReady(func() {
		s.win = mygo.NewWindow(mygo.WindowOptions{Title: "MyGo鲸屿 Launcher", Width: 1040, Height: 740, Content: ui.View(s.view)})
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
func (s *state) operation(work func(context.Context) error) { s.operationMode(work, true) }
func (s *state) operationMode(work func(context.Context) error, cancellable bool) {
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
		if err != nil {
			s.problem = err.Error()
		} else if probeErr != nil && !errors.Is(probeErr, externallauncher.ErrNotInstalled) {
			s.problem = probeErr.Error()
		}
		s.mu.Unlock()
		s.notify()
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
	s.operation(func(ctx context.Context) error {
		rows, err := update.ListNativeReleases(ctx, "github")
		if err != nil {
			return err
		}
		s.mu.Lock()
		version := s.installed.Version
		s.mu.Unlock()
		rows = update.PrepareNativeReleases(rows, version, update.NativeInstallerCacheDir(filepath.Join(paths.UserDataDir(), "updates")))
		s.mu.Lock()
		s.rows = rows
		s.message = "Choose a native desktop package to download and verify."
		s.mu.Unlock()
		return nil
	})
}
func (s *state) stage(id string) {
	s.operation(func(ctx context.Context) error {
		rows, err := update.ListNativeReleases(ctx, "github")
		if err != nil {
			return err
		}
		s.mu.Lock()
		version := s.installed.Version
		missing := s.installed.Executable == ""
		s.mu.Unlock()
		cache := update.NativeInstallerCacheDir(filepath.Join(paths.UserDataDir(), "updates"))
		rows = update.PrepareNativeReleases(rows, version, cache)
		selected, err := update.SelectNativeRelease(rows, id, runtime.GOOS, runtime.GOARCH, missing)
		if err != nil {
			return err
		}
		base := filepath.Join(paths.UserDataDir(), "updates")
		if err = os.MkdirAll(base, 0700); err != nil {
			return err
		}
		dir, err := os.MkdirTemp(base, "launcher-native-")
		if err != nil {
			return err
		}
		staged, err := update.StageNativeReleaseWithOptions(ctx, selected, update.NativeStageOptions{CurrentVersion: version, CacheDir: cache, RequireRuntime: missing, DestDir: dir, OnProgress: func(received, total int64) {
			s.mu.Lock()
			s.received = received
			s.total = total
			s.mu.Unlock()
			s.notify()
		}})
		if err != nil {
			_ = os.RemoveAll(dir)
			return err
		}
		s.mu.Lock()
		s.rows = rows
		s.staged = staged
		s.stagedVersion = selected.Version
		s.selected = selected.PackageKey()
		s.message = "Installer verified. Stop the desktop first, then install. This launcher remains open."
		s.mu.Unlock()
		return nil
	})
}
func (s *state) install() {
	s.mu.Lock()
	staged, version, busy := s.staged, s.stagedVersion, s.busy
	s.mu.Unlock()
	if busy || staged.Path == "" {
		return
	}
	choice, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: s.win, Type: mygo.MessageWarning, Title: "Install native desktop", Message: fmt.Sprintf("Install verified desktop %s? The desktop must be stopped first. The installer will ask for its destination. This launcher waits and adopts the registered installation; it does not quit.", version), Buttons: []string{"Cancel", "Install"}, DefaultButton: 0, CancelButton: 0})
	if err != nil || choice.Button != 1 {
		return
	}
	s.operationMode(func(ctx context.Context) error {
		fresh, err := s.controller.Install(ctx, staged, version)
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.installed = fresh
		s.staged = update.ApplyResult{}
		s.stagedVersion = ""
		s.message = "Installation registered and adopted. Start desktop when ready."
		s.mu.Unlock()
		return nil
	}, false)
}
func (s *state) view(c *ui.Context) {
	s.mu.Lock()
	m := launcher.Model{ThemeID: "light", Version: buildVersion, LauncherPackage: true, ManagedDesktop: true, VendorReady: s.installed.Executable != "", DesktopRunning: s.running, StartBusy: s.busy, StartError: s.problem, StatusNote: s.message, ActiveTab: s.tab, Releases: append([]update.NativeRelease(nil), s.rows...), ReleaseBusy: s.busy, ReleaseError: s.problem, ReleaseStatus: s.message, ReleaseSelectedPackage: s.selected, ReleaseStagedTag: s.stagedVersion, ReleaseReceived: s.received, ReleaseTotal: s.total}
	if s.installed.Version != "" {
		m.Version = s.installed.Version
	}
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		m.OnCancelRelease = cancel
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
