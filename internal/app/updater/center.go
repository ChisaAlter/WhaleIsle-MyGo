package updater

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/ui/i18n"
)

const (
	backgroundFirstCheck   = time.Minute
	backgroundInterval     = 6 * time.Hour
	backgroundNetworkRetry = 15 * time.Minute
)

// ReleaseState holds the updater's full native release download and staging state.
type ReleaseState struct {
	ModalOwner          WindowRef
	ModalParent         WindowRef
	ModalTemporary      bool
	ModalOff            func()
	ModalOpen           bool
	ReadyNotified       string
	ReadyNotice         NoticeHandle
	ReadyNoticeOff      func()
	Initialized         bool
	AttentionGeneration uint64
	Route               string
	CatalogRoute        string
	Rows                []update.NativeRelease
	Selected            string
	Busy                bool
	Installing          bool
	Phase               string
	Cancel              context.CancelFunc
	Status              string
	Err                 string
	Received, Total     int64
	Staged              update.ApplyResult
	StagedTag           string
	StagedTitle         string
	CenterLoaded        bool
	CenterLoading       bool
	AutoDownload        bool
	Paused              bool
	PauseRequested      bool
	CancelRequested     bool
	Active              update.NativeRelease
	Staging             string
	RequireRuntime      bool
	BackgroundCancel    context.CancelFunc
	BackgroundSequence  uint64
	ExplicitVersion     bool
	InstallRequested    bool
}

// AboutSnapshot is a read-only view of updater status for Settings > About.
type AboutSnapshot struct {
	UpdateBusy     bool
	UpdateStatus   string
	UpdatePhase    string
	UpdateVersion  string
	UpdatePercent  float64
	ActiveTag      string
	StagedPath     string
	StagedTag      string
	CenterLoaded   bool
	CenterLoading  bool
	AutoDownload   bool
	RequireRuntime bool
	Checking       bool
	Installing     bool
}

// LauncherSnapshot is a read-only value copy consumed by launcher UI wiring.
type LauncherSnapshot struct {
	NeedsRefresh        bool
	Rows                []update.NativeRelease
	RouteUnavailable    map[string]string
	Busy                bool
	Installing          bool
	Status              string
	Err                 string
	Selected            string
	Received, Total     int64
	Phase               string
	Paused              bool
	AutoDownload        bool
	StagedTag           string
	StagedTitle         string
	UpdateAvailable     bool
	UpdateStatus        string
	PendingConfirmation bool
}

// TraySnapshot is a read-only summary for the desktop tray menu.
type TraySnapshot struct {
	UpdateReady bool
	Title       string
}

// FrameSnapshot summarizes updater progress for main-frame badges/status bars.
type FrameSnapshot struct {
	Installing bool
	Phase      string
	Detail     string
	Busy       bool
	Checking   bool
}

// ModalSnapshot describes whether the native update center modal is active.
type ModalSnapshot struct {
	Open  bool
	Owner WindowRef
}

// Center owns native release discovery, paused/resumed downloads, background
// checks, startup update attention, legacy feed checks, and disk persistence.
type Center struct {
	bridge       ShellBridge
	buildVersion string

	diskMu sync.Mutex // Acquired strictly before mu; never inside mu.
	mu     sync.Mutex

	releases      ReleaseState
	checkPending  bool
	progress      *ProgressState
	attention     attentionState
	notifyAt      atomic.Int64
	fallbackRoute string

	journal journalBook
}

// NewCenter constructs an updater Center bound to an optional ShellBridge.
func NewCenter(bridge ShellBridge) *Center {
	return &Center{bridge: bridge}
}

// NewCenterForTest constructs an updater Center with explicit build version and route defaults.
func NewCenterForTest(bridge ShellBridge, buildVersion, defaultRoute string) *Center {
	return &Center{
		bridge:        bridge,
		buildVersion:  buildVersion,
		fallbackRoute: defaultRoute,
	}
}

// SetBuildVersion overrides the build version when no ShellBridge provides one.
func (c *Center) SetBuildVersion(version string) {
	c.mu.Lock()
	c.buildVersion = version
	c.mu.Unlock()
}

// SetFallbackRoute overrides the fallback download route when no ShellBridge is bound.
func (c *Center) SetFallbackRoute(route string) {
	c.mu.Lock()
	c.fallbackRoute = route
	c.mu.Unlock()
}

// RestoreReleaseState seeds initial release state (primarily for tests before EnsureLoaded).
func (c *Center) RestoreReleaseState(r ReleaseState) {
	c.mu.Lock()
	c.releases = r
	c.mu.Unlock()
}

// SetVendorReady updates whether the bundled DSH vendor runtime is present.
func (c *Center) SetVendorReady(ready bool) {
	c.mu.Lock()
	c.attention.vendorReady = ready
	c.mu.Unlock()
}

// Hold reports whether startup launcher auto-close is currently held by update review.
func (c *Center) Hold() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.attention.hold
}

// SetHold updates whether launcher closing is held for update confirmation/review.
func (c *Center) SetHold(hold bool) {
	c.mu.Lock()
	c.attention.hold = hold
	c.mu.Unlock()
}

// ReleaseSnapshot returns a consistent copy of the current native release state.
func (c *Center) ReleaseSnapshot() ReleaseState {
	c.mu.Lock()
	defer c.mu.Unlock()
	snap := c.releases
	snap.Rows = append([]update.NativeRelease(nil), snap.Rows...)
	return snap
}

// TraySnapshot returns tray-menu readiness summary under lock.
func (c *Center) TraySnapshot() TraySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	tag := c.releases.StagedTag
	title := c.releases.StagedTitle
	if title == "" {
		title = tag
	}
	return TraySnapshot{UpdateReady: tag != "", Title: title}
}

// FrameSnapshot returns a lightweight status summary for frame/header consumers.
func (c *Center) FrameSnapshot() FrameSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return FrameSnapshot{
		Installing: c.releases.Installing,
		Phase:      c.releases.Phase,
		Detail:     strings.TrimSpace(c.releases.Status + " " + c.releases.Err),
		Busy:       c.releases.Busy || c.checkPending,
		Checking:   c.checkPending,
	}
}

// ModalSnapshot returns the current modal open state and owning window.
func (c *Center) ModalSnapshot() ModalSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return ModalSnapshot{Open: c.releases.ModalOpen, Owner: c.releases.ModalOwner}
}

// SetCatalogStatus updates catalog display status/error fields (used for testing).
func (c *Center) SetCatalogStatus(status, errMsg string) {
	c.mu.Lock()
	c.releases.Status = status
	c.releases.Err = errMsg
	c.mu.Unlock()
}

// AboutSnapshot returns the Settings > About state, initializing from disk first.
func (c *Center) AboutSnapshot(win WindowRef) AboutSnapshot {
	if win != nil && (c.bridge == nil || c.bridge.WindowAlive(win)) {
		c.EnsureLoadedAsync(win)
	} else {
		c.EnsureLoaded()
	}
	ver := c.env().BuildVersion
	c.mu.Lock()
	releases := c.releases
	rows := append([]update.NativeRelease(nil), releases.Rows...)
	checking := c.checkPending
	c.mu.Unlock()

	snap := AboutSnapshot{
		UpdateBusy:     releases.Busy || checking || releases.CenterLoading,
		UpdateStatus:   strings.TrimSpace(releases.Status),
		UpdatePhase:    releases.Phase,
		ActiveTag:      releases.Active.Tag,
		StagedPath:     releases.Staged.Path,
		StagedTag:      releases.StagedTag,
		CenterLoaded:   releases.CenterLoaded,
		CenterLoading:  releases.CenterLoading,
		AutoDownload:   releases.AutoDownload,
		RequireRuntime: releases.RequireRuntime,
		Checking:       checking,
		Installing:     releases.Installing,
	}
	if message := strings.TrimSpace(releases.Err); message != "" && !strings.Contains(snap.UpdateStatus, message) {
		snap.UpdateStatus = strings.TrimSpace(snap.UpdateStatus + "\n" + message)
	}
	if checking && (snap.UpdatePhase == "" || snap.UpdatePhase == "idle") {
		snap.UpdatePhase = "checking"
	}
	if releases.Installing {
		snap.UpdatePhase = "installing"
	}
	for _, row := range rows {
		if row.UnavailableReason == "" && !row.Prerelease && update.CompareVersions(row.Version, ver) > 0 {
			snap.UpdateVersion = row.Version
			if snap.UpdatePhase == "" || snap.UpdatePhase == "idle" {
				snap.UpdatePhase = "available"
			}
			break
		}
	}
	if snap.UpdateVersion == "" && releases.StagedTag != "" {
		snap.UpdateVersion = releases.StagedTag
	}
	if releases.Total > 0 && releases.Received >= 0 {
		snap.UpdatePercent = float64(releases.Received) * 100 / float64(releases.Total)
	}
	return snap
}

// PrepareLauncherModel triggers lazy catalog loading/attention and builds a snapshot
// matching what launcher/modal views need.
func (c *Center) PrepareLauncherModel(win WindowRef, vendorReady bool) LauncherSnapshot {
	c.EnsureLoadedAsync(win)
	env := c.env()
	route := env.DownloadRoute

	c.mu.Lock()
	c.attention.vendorReady = vendorReady
	refresh := c.releases.CenterLoaded && !c.releases.Busy && (!c.releases.Initialized || c.releases.Route != route)
	snapshot := c.releases
	snapshot.Rows = append([]update.NativeRelease(nil), snapshot.Rows...)
	c.mu.Unlock()

	if refresh {
		c.Refresh(win)
	}

	out := LauncherSnapshot{
		NeedsRefresh:     refresh,
		Rows:             snapshot.Rows,
		RouteUnavailable: c.RouteAvailability(),
		Busy:             snapshot.Busy || refresh,
		Installing:       snapshot.Installing,
		Status:           snapshot.Status,
		Err:              snapshot.Err,
		Selected:         snapshot.Selected,
		Received:         snapshot.Received,
		Total:            snapshot.Total,
		Phase:            snapshot.Phase,
		Paused:           snapshot.Paused,
		AutoDownload:     snapshot.AutoDownload,
		StagedTag:        snapshot.StagedTag,
		StagedTitle:      snapshot.StagedTitle,
	}
	if out.StagedTitle == "" {
		out.StagedTitle = snapshot.StagedTag
	}
	for _, row := range snapshot.Rows {
		if row.UnavailableReason == "" && (vendorReady || row.RuntimeIncluded) && !row.Prerelease && update.CompareVersions(row.Version, env.BuildVersion) > 0 {
			out.UpdateAvailable = true
			out.UpdateStatus = fmt.Sprintf(i18n.T(i18n.KeyCmdMygoIsAvailable), row.Version)
			break
		}
	}
	c.MaybeAskStartupUpdate(win, snapshot, vendorReady)
	c.mu.Lock()
	pendingUpdate := c.attention.tracker.Pending()
	c.mu.Unlock()
	out.PendingConfirmation = pendingUpdate
	if pendingUpdate && out.UpdateAvailable {
		out.UpdateStatus += i18n.T(i18n.KeyCmdAwaitingConfirmationChooseLaterOrDownl)
	}
	return out
}

// SelectRelease sets the active selected package key when idle.
func (c *Center) SelectRelease(win WindowRef, packageID string) {
	c.mu.Lock()
	if !c.releases.Busy {
		c.releases.Selected = packageID
	}
	c.mu.Unlock()
	if c.bridge != nil && win != nil {
		c.bridge.UpdateWindow(win, func() {})
	}
}

// DismissRelease clears transient status/error banner text and recomputes phase.
func (c *Center) DismissRelease(win WindowRef) {
	ver := c.env().BuildVersion
	c.mu.Lock()
	r := &c.releases
	r.Err = ""
	r.Status = ""
	r.Phase = settledNativeReleasePhase(*r, ver)
	c.mu.Unlock()
	c.Notify(win)
}

// BackgroundActive reports whether background polling goroutine currently holds a cancel handle.
func (c *Center) BackgroundActive() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.releases.BackgroundCancel != nil
}

// StartBackground binds scheduled native release polling to the application lifecycle.
func (c *Center) StartBackground(win WindowRef) {
	if c == nil || AutoUpdateDisabled() {
		return
	}
	c.EnsureLoadedAsync(win)
	env := c.env()
	if env.Quitting {
		return
	}
	c.mu.Lock()
	r := &c.releases
	if r.BackgroundCancel != nil {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(c.lifecycleContext())
	r.BackgroundSequence++
	sequence := r.BackgroundSequence
	r.BackgroundCancel = cancel
	c.mu.Unlock()
	go func() {
		defer cancel()
		defer func() {
			c.mu.Lock()
			if c.releases.BackgroundSequence == sequence {
				c.releases.BackgroundCancel = nil
			}
			c.mu.Unlock()
		}()
		delay := backgroundFirstCheck
		for {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			delay = backgroundInterval
			settled := make(chan error, 1)
			if !c.RefreshOwned(win, ctx, func(err error) { settled <- err }) {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case err := <-settled:
				if nativeUpdateNetworkError(err) {
					delay = backgroundNetworkRetry
				}
			}
		}
	}()
}

// StopBackground cancels scheduled checks and pauses an active download while
// leaving an in-progress installer hand-off intact.
func (c *Center) StopBackground() {
	if c == nil {
		return
	}
	c.ClearReadyNotice()
	c.CloseModal(nil)
	c.mu.Lock()
	r := &c.releases
	r.BackgroundSequence++
	background, active := r.BackgroundCancel, r.Cancel
	r.BackgroundCancel = nil
	if r.Busy && !r.Installing && r.Phase == "downloading" {
		r.PauseRequested = true
	}
	if r.Installing {
		active = nil
	}
	c.mu.Unlock()
	if background != nil {
		background()
	}
	if active != nil {
		active()
	}
}

// Refresh starts an asynchronous native release catalog check if idle.
func (c *Center) Refresh(win WindowRef) {
	c.RefreshOwned(win, c.lifecycleContext(), nil)
}

// RefreshOwned starts a catalog lookup under parent context if admissible.
func (c *Center) RefreshOwned(win WindowRef, parent context.Context, settled func(error)) bool {
	if c == nil {
		return false
	}
	c.EnsureLoaded()
	env := c.env()
	route := env.DownloadRoute
	c.mu.Lock()
	if c.releases.Busy || env.Quitting || env.LauncherImportBusy || env.LauncherStartBusy || env.TaskRestartPending || parent.Err() != nil {
		c.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithCancel(parent)
	c.releases.AttentionGeneration++
	c.releases.Initialized = true
	c.releases.Route = route
	c.releases.Busy, c.releases.Cancel = true, cancel
	c.releases.Status = i18n.T(i18n.KeyCmdLoadingMygoReleaseCatalog)
	c.releases.Phase = "checking"
	c.releases.Err = ""
	c.releases.Received, c.releases.Total = 0, 0
	c.mu.Unlock()

	userDataDir := c.userDataDir()
	buildVer := env.BuildVersion
	go func() {
		defer cancel()
		rows, err := update.ListNativeReleases(ctx, route)
		if err == nil {
			rows = update.PrepareNativeReleases(rows, buildVer, update.NativeInstallerCacheDir(filepath.Join(userDataDir, "updates")))
		}
		c.mu.Lock()
		r := &c.releases
		r.Busy, r.Cancel = false, nil
		if err != nil {
			r.Err = err.Error()
			r.Status = i18n.T(i18n.KeyCmdCatalogLookupFailedThePreviousListAnd)
			if errors.Is(err, context.Canceled) {
				r.Status = i18n.T(i18n.KeyCmdReleaseLookupCancelled)
			}
		} else {
			requireRuntime := !c.attention.vendorReady
			for i := range rows {
				if rows[i].UnavailableReason == "" {
					if verr := rows[i].Validate(runtime.GOOS, runtime.GOARCH, requireRuntime); verr != nil {
						rows[i].UnavailableReason = verr.Error()
					}
				}
			}
			r.Rows = rows
			r.CatalogRoute = route
			r.Status = fmt.Sprintf(i18n.T(i18n.KeyCmdLoadedMygoReleases), len(rows))
			if len(rows) == 0 {
				r.Status = i18n.T(i18n.KeyCmdNoCompatibleMygoInstallerHasBeenPublis)
			}
			found := false
			for _, row := range rows {
				if row.PackageKey() == r.Selected && row.UnavailableReason == "" {
					found = true
					break
				}
			}
			if !found {
				r.Selected = ""
				for _, row := range rows {
					if row.UnavailableReason == "" {
						r.Selected = row.PackageKey()
						break
					}
				}
			}
		}
		r.Phase = settledNativeReleasePhase(*r, buildVer)
		c.mu.Unlock()
		c.Notify(win)
		c.MaybeAutoDownload(win)
		if settled != nil {
			settled(err)
		}
	}()
	c.Notify(win)
	return true
}

// Download begins downloading, verifying, and staging the selected or newest compatible release.
func (c *Center) Download(win WindowRef, tag string, requireRuntime bool, expected ...startuprec.UpdateAttentionScope) {
	if c == nil {
		return
	}
	c.EnsureLoaded()
	env := c.env()
	ownedWin, winKey := c.windowContext(win)
	route := env.DownloadRoute
	userDataDir := c.userDataDir()
	buildVer := env.BuildVersion

	c.mu.Lock()
	r := &c.releases
	if len(expected) > 0 {
		scope, _ := c.startupUpdateScopeLocked(env, ownedWin, win, winKey, *r, !requireRuntime)
		if scope != expected[0] || route != expected[0].Route || tag != expected[0].Tag || !env.AskOnUpdate {
			c.mu.Unlock()
			return
		}
		c.attention.hold = true
	}
	if r.Busy || env.Quitting || env.LauncherImportBusy || env.LauncherStartBusy || env.TaskRestartPending {
		c.mu.Unlock()
		return
	}
	if requireRuntime && strings.TrimSpace(os.Getenv(paths.EnvVendor)) != "" {
		r.Err = i18n.T(i18n.KeyCmdWhaleisleVendorOverridesTheRuntimePath)
		r.Phase = "error"
		c.mu.Unlock()
		c.Notify(win)
		return
	}
	rows := append([]update.NativeRelease(nil), r.Rows...)
	var pinned *update.NativeRelease
	if r.Paused && tag == r.Active.PackageKey() && r.Active.Tag != "" {
		cp := r.Active
		pinned = &cp
	}
	if r.CatalogRoute != route {
		rows = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.AttentionGeneration++
	c.attention.tracker.DismissCurrent()
	r.Initialized, r.Route, r.Busy, r.Cancel = true, route, true, cancel
	r.Status, r.Err = i18n.T(i18n.KeyCmdSelectingAndVerifyingAMygoInstaller), ""
	r.Paused, r.PauseRequested, r.CancelRequested = false, false, false
	r.Phase = ""
	if len(rows) == 0 {
		r.Phase = "checking"
	}
	r.Received, r.Total = 0, 0
	c.mu.Unlock()

	go func() {
		defer cancel()
		var selected update.NativeRelease
		var staged update.ApplyResult
		var staging string
		err := func() error {
			if len(rows) == 0 && pinned == nil {
				var err error
				rows, err = update.ListNativeReleases(ctx, route)
				if err != nil {
					return err
				}
			}
			cacheDir := update.NativeInstallerCacheDir(filepath.Join(userDataDir, "updates"))
			rows = update.PrepareNativeReleases(rows, buildVer, cacheDir)
			var selectErr error
			if pinned != nil {
				selected = *pinned
				selectErr = selected.Validate(runtime.GOOS, runtime.GOARCH, requireRuntime)
			} else {
				selected, selectErr = update.SelectNativeRelease(rows, tag, runtime.GOOS, runtime.GOARCH, requireRuntime)
			}
			if selectErr != nil {
				return selectErr
			}
			c.mu.Lock()
			c.releases.Selected = selected.PackageKey()
			c.releases.Active = selected
			c.releases.RequireRuntime = requireRuntime
			if pinned == nil {
				c.releases.ExplicitVersion = tag != ""
			}
			c.releases.InstallRequested = false
			c.releases.Staged, c.releases.StagedTag, c.releases.StagedTitle = update.ApplyResult{}, "", ""
			c.releases.Status = i18n.T(i18n.KeyCmdDownloadingAndVerifyingFullInstaller) + selected.Tag
			if selected.IsDelta() {
				c.releases.Status = i18n.T(i18n.KeyCmdReusingCachedInstallerAndDownloadingCh) + selected.Tag
			}
			c.mu.Unlock()

			c.RecordAction(update.JournalActionDownloadRequested)
			startPercent := 0.0
			c.RecordState(update.JournalState{Phase: update.JournalPhaseDownloading, Version: selected.Version, Percent: &startPercent})
			base := filepath.Join(userDataDir, "updates")
			if err := os.MkdirAll(base, 0755); err != nil {
				return err
			}
			staging = nativeReleaseStagingDir(userDataDir, selected)
			if err := os.MkdirAll(staging, 0755); err != nil {
				return err
			}
			c.mu.Lock()
			c.releases.Staging = staging
			c.mu.Unlock()
			c.Persist()
			var err error
			staged, err = update.StageNativeReleaseWithOptions(ctx, selected, update.NativeStageOptions{
				Resume:         true,
				CurrentVersion: buildVer,
				CacheDir:       cacheDir,
				RequireRuntime: requireRuntime,
				DestDir:        staging,
				OnPhase: func(phase update.StagePhase) {
					if phase == update.StageReady {
						// The returned ApplyResult must be durably committed with
						// ready; a progress phase alone has no installer identity.
						return
					}
					c.mu.Lock()
					c.releases.Phase = string(phase)
					c.mu.Unlock()
					c.Persist()
					c.Notify(win)
				},
				OnProgress: func(received, total int64) {
					c.mu.Lock()
					previous := DownloadPercent(c.releases.Received, c.releases.Total)
					c.releases.Received, c.releases.Total = received, total
					c.mu.Unlock()
					percent := DownloadPercent(received, total)
					if int(previous) != int(percent) {
						c.Persist()
					}
					c.RecordState(update.JournalState{Phase: update.JournalPhaseDownloading, Version: selected.Version, Percent: &percent})
					c.NotifyProgress(win)
				},
			})
			if err != nil {
				return err
			}
			return ctx.Err()
		}()

		journal := c.completeDownload(userDataDir, buildVer, route, rows, selected, staged, err)
		if journal.Phase != "" {
			c.RecordState(journal)
		}
		c.Notify(win)
	}()
	c.Notify(win)
}

// Pause requests cancellation of an active download while marking it resumable.
func (c *Center) Pause(win WindowRef) {
	if c == nil {
		return
	}
	c.mu.Lock()
	r := &c.releases
	if r.Phase != "downloading" || !r.Busy {
		c.mu.Unlock()
		return
	}
	r.PauseRequested = true
	cancel := r.Cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.Notify(win)
}

// Resume restarts a previously paused native update download.
func (c *Center) Resume(win WindowRef) {
	if c == nil {
		return
	}
	c.EnsureLoaded()
	c.mu.Lock()
	r := &c.releases
	if r.Busy || r.Active.Tag == "" {
		c.mu.Unlock()
		return
	}
	r.Paused = true
	key, requireRuntime := r.Active.PackageKey(), r.RequireRuntime
	c.mu.Unlock()
	c.Download(win, key, requireRuntime)
}

// Cancel aborts a download or clears staged installer state unless installation is in progress.
func (c *Center) Cancel(win WindowRef) {
	if c == nil {
		return
	}
	ver := c.env().BuildVersion
	c.mu.Lock()
	r := &c.releases
	if r.Installing {
		c.mu.Unlock()
		return
	}
	if r.Busy {
		r.CancelRequested = true
		r.PauseRequested = false
		cancel := r.Cancel
		c.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return
	}
	dir := r.Staging
	r.Active = update.NativeRelease{}
	r.Staged, r.StagedTag, r.StagedTitle = update.ApplyResult{}, "", ""
	r.Staging = ""
	r.Paused = false
	r.Received, r.Total = 0, 0
	r.Err = ""
	r.Status = lifecycleText("下载已取消。", "Download cancelled.")
	r.Phase = settledNativeReleasePhase(*r, ver)
	c.mu.Unlock()
	if dir != "" {
		c.diskMu.Lock()
		discardErr := DiscardNativePartial(c.userDataDir(), dir)
		c.diskMu.Unlock()
		if discardErr != nil {
			c.mu.Lock()
			if !c.releases.Busy && c.releases.Active.Tag == "" && c.releases.Staged.Path == "" {
				c.releases.Err = discardErr.Error()
				c.releases.Phase = "error"
			}
			c.mu.Unlock()
		}
	}
	c.Persist()
	c.Notify(win)
}

// SetAutoDownload updates the persistent automatic download preference.
func (c *Center) SetAutoDownload(win WindowRef, enabled bool) {
	if c == nil {
		return
	}
	c.EnsureLoaded()
	if c.bridge != nil {
		if err := c.bridge.SetAutoDownloadPref(enabled); err != nil {
			c.mu.Lock()
			c.releases.Err = lifecycleText("保存自动下载偏好失败：", "Could not save automatic download preference: ") + err.Error()
			c.mu.Unlock()
			c.Notify(win)
			return
		}
	}
	c.mu.Lock()
	c.releases.AutoDownload = enabled
	c.mu.Unlock()
	c.Persist()
	if enabled {
		c.MaybeAutoDownload(win)
	}
	c.Notify(win)
}

// MaybeAutoDownload starts downloading the newest eligible release when enabled and idle.
func (c *Center) MaybeAutoDownload(win WindowRef) {
	if c == nil || AutoUpdateDisabled() {
		return
	}
	env := c.env()
	c.mu.Lock()
	r := c.releases
	requireRuntime := !c.attention.vendorReady
	allowed := r.AutoDownload && !r.Busy && !r.Paused && r.Err == "" && r.CatalogRoute == r.Route && r.Staged.Path == "" && !env.Quitting && !env.LauncherImportBusy && !env.LauncherStartBusy && !env.TaskRestartPending
	available := false
	if allowed {
		for _, release := range r.Rows {
			if !release.Prerelease && release.UnavailableReason == "" && (!requireRuntime || release.RuntimeIncluded) && update.CompareVersions(release.Version, env.BuildVersion) > 0 {
				available = true
				break
			}
		}
	}
	c.mu.Unlock()
	if available {
		c.Download(win, "", requireRuntime)
	}
}

// StartStaged validates the staged installer and triggers protected installation with user consent.
func (c *Center) StartStaged(win WindowRef) {
	if c == nil {
		return
	}
	env := c.env()
	c.mu.Lock()
	staged, busy := c.releases.Staged, c.releases.Busy
	c.mu.Unlock()
	if busy || staged.Path == "" {
		return
	}
	if !update.CanStartStaged(staged) {
		c.mu.Lock()
		c.releases.Phase = "error"
		c.releases.Err = i18n.T(i18n.KeyCmdThisPlatformCannotStartTheInstaller)
		c.mu.Unlock()
		c.Notify(win)
		return
	}
	c.mu.Lock()
	if c.releases.Busy || c.releases.Staged != staged || c.releases.Staged.Path == "" || env.TaskRestartPending || env.Quitting || env.LauncherImportBusy {
		c.mu.Unlock()
		return
	}
	c.releases.Busy = true
	c.releases.Installing = true
	c.releases.Phase = "installing"
	c.mu.Unlock()

	buildVer := env.BuildVersion
	go func() {
		defer func() {
			c.mu.Lock()
			c.releases.Busy = false
			c.releases.Installing = false
			c.releases.Phase = settledNativeReleasePhase(c.releases, buildVer)
			c.mu.Unlock()
			c.Notify(win)
		}()
		if c.bridge == nil || !c.bridge.InstallStagedWithConsent(win, staged, true) {
			c.mu.Lock()
			c.releases.Status = i18n.T(i18n.KeyCmdVerifiedInstallerRetainedForRetryTheCu)
			c.mu.Unlock()
			c.Persist()
			c.Notify(win)
		}
	}()
	c.Notify(win)
}

// MarkInstallerLaunched records that the OS installer process was actually launched.
func (c *Center) MarkInstallerLaunched(staged update.ApplyResult) {
	if c == nil {
		return
	}
	c.mu.Lock()
	r := &c.releases
	owned := r.Staged.Path == staged.Path && staged.Path != "" && r.Staged.SHA256 == staged.SHA256 && r.Staged.Size == staged.Size
	if owned {
		r.InstallRequested = true
	}
	c.mu.Unlock()
	if owned {
		c.Persist()
	}
}

// NotifyProgress throttles window repaints during fast byte streaming.
func (c *Center) NotifyProgress(win WindowRef) {
	if c == nil {
		return
	}
	now := time.Now().UnixNano()
	last := c.notifyAt.Load()
	if now-last < int64(120*time.Millisecond) {
		return
	}
	if !c.notifyAt.CompareAndSwap(last, now) {
		return
	}
	c.Notify(win)
}

// LastProgressNotifyAt returns the UnixNano timestamp of the most recent throttled repaint.
func (c *Center) LastProgressNotifyAt() int64 {
	if c == nil {
		return 0
	}
	return c.notifyAt.Load()
}

// Notify refreshes tray status and updates owning windows on the UI thread.
func (c *Center) Notify(win WindowRef) {
	if c == nil || c.bridge == nil {
		return
	}
	c.bridge.RefreshTray()
	mainWin, lwWin := c.bridge.PrimaryWindows()
	target := win
	if target == nil || !c.bridge.WindowAlive(target) {
		target = mainWin
		if lwWin != nil && c.bridge.WindowAlive(lwWin) {
			target = lwWin
		}
	}
	if target == nil || !c.bridge.WindowAlive(target) {
		return
	}
	c.bridge.UpdateWindow(target, func() {
		owner := c.bridge.DialogOwner(win)
		if owner == nil {
			mWin, lWin := c.bridge.PrimaryWindows()
			owner = mWin
			if lWin != nil && c.bridge.WindowAlive(lWin) {
				owner = lWin
			}
		}
		if owner != nil && c.bridge.WindowAlive(owner) {
			c.RefreshAttention(owner)
			c.NotifyReady(owner)
		}
		mWin, lWin := c.bridge.PrimaryWindows()
		if mWin != nil && mWin != target && c.bridge.WindowAlive(mWin) {
			c.bridge.InvalidateWindow(mWin)
		}
		if lWin != nil && c.bridge.WindowAlive(lWin) && lWin != target {
			c.bridge.InvalidateWindow(lWin)
		}
	})
}

func nativeUpdateNetworkError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var network net.Error
	var status *update.CatalogHTTPError
	return errors.As(err, &network) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &status) && status.StatusCode >= 500
}

func settledNativeReleasePhase(r ReleaseState, buildVersion string) string {
	if r.Paused {
		return "paused"
	}
	if r.Err != "" {
		return "error"
	}
	if r.Staged.Path != "" {
		return "ready"
	}
	for _, row := range r.Rows {
		if row.UnavailableReason == "" && !row.Prerelease && update.CompareVersions(row.Version, buildVersion) > 0 {
			return "available"
		}
	}
	return ""
}

func nativeStagedTitle(release update.NativeRelease) string {
	variant := i18n.T(i18n.KeyCmdDesktopOnly)
	if release.RuntimeIncluded {
		variant = i18n.T(i18n.KeyCmdIncludesRuntime)
	}
	kind := i18n.T(i18n.KeyCmdFullPackage)
	if release.IsDelta() {
		kind = i18n.T(i18n.KeyCmdDeltaReconstruction)
	}
	return release.Tag + " · " + kind + " · " + variant
}

// RouteAvailability explains why a download source cannot be selected.
func (c *Center) RouteAvailability() map[string]string {
	unavailable := make(map[string]string)
	if _, err := update.NativeCatalogURL("cnb"); err != nil {
		unavailable["cnb"] = lifecycleText("CNB 尚未配置 鲸屿 MyGo 原生发布源，请使用 GitHub。", "CNB has no configured Whale Isle MyGo native release source. Use GitHub.")
	}
	return unavailable
}
