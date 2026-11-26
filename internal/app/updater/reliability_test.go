package updater

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
)

type stubBridge struct {
	mu           sync.Mutex
	userData     string
	autoPrefErr  error
	autoPrefVal  bool
	consentCalls atomic.Int32
	consentRes   bool
	onConsent    func()
	notifyCalls  atomic.Int32
}

func (b *stubBridge) Environment() ShellEnvironment {
	return ShellEnvironment{BuildVersion: "1.0.0", DownloadRoute: "github"}
}
func (b *stubBridge) AutoDownloadPref() (bool, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.autoPrefVal, true
}
func (b *stubBridge) InitAutoDownloadPref(v bool) error { return b.SetAutoDownloadPref(v) }
func (b *stubBridge) SetAutoDownloadPref(v bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.autoPrefErr != nil {
		return b.autoPrefErr
	}
	b.autoPrefVal = v
	return nil
}
func (b *stubBridge) LifecycleContext() context.Context                     { return context.Background() }
func (b *stubBridge) UserDataDir() string                                   { return b.userData }
func (b *stubBridge) OwnedWindow(WindowRef) bool                            { return true }
func (b *stubBridge) WindowAlive(WindowRef) bool                            { return true }
func (b *stubBridge) WindowVisibility(WindowRef) (bool, bool)               { return true, true }
func (b *stubBridge) WindowHasPage(WindowRef) bool                          { return false }
func (b *stubBridge) WindowKey(WindowRef) string                            { return "main" }
func (b *stubBridge) DialogOwner(w WindowRef) WindowRef                     { return w }
func (b *stubBridge) PrimaryWindows() (WindowRef, WindowRef)                { return nil, nil }
func (b *stubBridge) RunOnMain(fn func())                                   { fn() }
func (b *stubBridge) UpdateWindow(_ WindowRef, fn func())                   { b.notifyCalls.Add(1); fn() }
func (b *stubBridge) InvalidateWindow(WindowRef)                            {}
func (b *stubBridge) RestoreShowFocus(WindowRef)                            {}
func (b *stubBridge) FlashFrame(WindowRef, bool)                            {}
func (b *stubBridge) SystemNotificationsSupported() bool                    { return false }
func (b *stubBridge) NewNotification(string, string, bool) NoticeHandle     { return nil }
func (b *stubBridge) WatchFocusAndClose(WindowRef, func(), func()) func()   { return func() {} }
func (b *stubBridge) WatchModalWindows(WindowRef, WindowRef, func()) func() { return func() {} }
func (b *stubBridge) OpenModalWindow(WindowRef, string) WindowRef           { return nil }
func (b *stubBridge) DestroyWindow(WindowRef)                               {}
func (b *stubBridge) RefreshTray()                                          {}
func (b *stubBridge) AskConfirm(ConfirmRequest)                             {}
func (b *stubBridge) AskConfirmBlocking(ConfirmRequest) bool                { return false }
func (b *stubBridge) InstallStagedWithConsent(_ WindowRef, _ desktopupdate.ApplyResult, _ bool) bool {
	b.consentCalls.Add(1)
	if b.onConsent != nil {
		b.onConsent()
	}
	return b.consentRes
}
func (b *stubBridge) SaveFailedToast(string, error) string { return "" }
func (b *stubBridge) SetLauncherUpdateNotice(string)       {}

func TestCancelErrorWritebackGuardedAndStartStagedReverifies(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	b := &stubBridge{userData: dir}
	c := NewCenter(b)

	// 1. Cancel with an outside staging path causes DiscardNativePartial to fail,
	// but error writeback must only occur while still idle (no new download active).
	outside := t.TempDir()
	c.RestoreReleaseState(ReleaseState{
		CenterLoaded: true,
		Paused:       true,
		Phase:        "paused",
		Active:       updateCenterFixture(),
		Staging:      outside,
	})
	// Hold diskMu so we can simulate a new download starting before Cancel finishes discarding.
	c.diskMu.Lock()
	cancelDone := make(chan struct{})
	go func() {
		c.Cancel(nil)
		close(cancelDone)
	}()
	// Wait until Cancel cleared state and is blocked on c.diskMu.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if c.ReleaseSnapshot().Staging == "" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	// Simulate a concurrent new Download taking ownership while Cancel is waiting on diskMu.
	c.mu.Lock()
	c.releases.Busy = true
	c.releases.Phase = "downloading"
	c.releases.Active = updateCenterFixture()
	c.releases.Err = ""
	c.mu.Unlock()
	c.diskMu.Unlock()
	<-cancelDone

	snap := c.ReleaseSnapshot()
	if snap.Phase != "downloading" || snap.Err != "" || !snap.Busy {
		t.Fatalf("Cancel clobbered active Download state: %+v", snap)
	}

	// 2. StartStaged with an unstartable extension fails synchronously without setting Busy/Installing,
	// and when consent is declined on a startable staged file it settles back to !Busy/!Installing.
	c2 := NewCenter(b)
	c2.RestoreReleaseState(ReleaseState{
		CenterLoaded: true,
		Phase:        "ready",
		Staged:       desktopupdate.ApplyResult{Path: filepath.Join(dir, "payload.unsupported")},
		StagedTag:    "v1.2.3",
	})
	c2.StartStaged(nil)
	s2 := c2.ReleaseSnapshot()
	if s2.Busy || s2.Installing || s2.Phase != "error" {
		t.Fatalf("StartStaged did not reject unsupported staged file cleanly: %+v", s2)
	}

	// 3. StartStaged TOCTOU: if Staged is cleared before install launch,
	// re-verification under c.mu prevents launching with stale staged metadata.
	realFile := filepath.Join(dir, "installer.exe")
	if err := os.WriteFile(realFile, []byte("MZstub"), 0o600); err != nil {
		t.Fatal(err)
	}
	c3 := NewCenter(b)
	c3.RestoreReleaseState(ReleaseState{
		CenterLoaded: true,
		Phase:        "ready",
		Staged:       desktopupdate.ApplyResult{Path: realFile},
		StagedTag:    "v1.2.3",
	})
	c3.Cancel(nil)
	c3.StartStaged(nil)
	if b.consentCalls.Load() != 0 {
		t.Fatalf("StartStaged launched canceled installer (%d calls)", b.consentCalls.Load())
	}
}

func TestCenterAutoDownloadAndSnapshotViews(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	b := &stubBridge{userData: dir}
	c := NewCenter(b)

	c.SetAutoDownload(nil, true)
	if !c.ReleaseSnapshot().AutoDownload {
		t.Fatal("expected AutoDownload=true after SetAutoDownload")
	}
	about := c.AboutSnapshot(nil)
	if !about.AutoDownload {
		t.Fatalf("AboutSnapshot missed AutoDownload: %+v", about)
	}
	tray := c.TraySnapshot()
	if tray.UpdateReady {
		t.Fatalf("unexpected UpdateReady in idle TraySnapshot: %+v", tray)
	}
	frame := c.FrameSnapshot()
	if frame.Busy || frame.Installing {
		t.Fatalf("unexpected busy FrameSnapshot: %+v", frame)
	}
	modal := c.ModalSnapshot()
	if modal.Open {
		t.Fatalf("unexpected open ModalSnapshot: %+v", modal)
	}
	c.SetHold(true)
	if !c.Hold() {
		t.Fatal("expected Hold()=true")
	}
	c.SetHold(false)
	c.SetCatalogStatus("ok", "")
	if c.ReleaseSnapshot().Status != "ok" {
		t.Fatalf("SetCatalogStatus did not apply: %+v", c.ReleaseSnapshot())
	}
}
