package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/ui/launcher"
)

type nativeReleaseState struct {
	initialized         bool
	attentionGeneration uint64
	route               string
	catalogRoute        string
	rows                []update.NativeRelease
	selected            string
	busy                bool
	installing          bool
	phase               string
	cancel              context.CancelFunc
	status              string
	err                 string
	received, total     int64
	staged              update.ApplyResult
	stagedTag           string
	stagedTitle         string
}

func (s *mainState) withNativeReleases(win *mygo.Window, m launcher.Model) launcher.Model {
	route := s.updateDownloadRoute()
	s.mu.Lock()
	s.startupRecovery.updateVendorReady = m.VendorReady
	refresh := !s.nativeReleases.busy && (!s.nativeReleases.initialized || s.nativeReleases.route != route)
	snapshot := s.nativeReleases
	snapshot.rows = append([]update.NativeRelease(nil), snapshot.rows...)
	s.mu.Unlock()
	if refresh {
		s.refreshNativeReleases(win)
	}
	m.Releases = snapshot.rows
	m.ReleaseBusy = snapshot.busy || refresh
	m.ReleaseStatus = snapshot.status
	m.ReleaseError = snapshot.err
	m.ReleaseSelectedPackage = snapshot.selected
	m.ReleaseReceived, m.ReleaseTotal = snapshot.received, snapshot.total
	m.ReleaseStagedTag = snapshot.stagedTitle
	if m.ReleaseStagedTag == "" {
		m.ReleaseStagedTag = snapshot.stagedTag
	}
	m.OnRefreshReleases = func() { s.refreshNativeReleases(win) }
	m.OnCheckUpdate = m.OnRefreshReleases
	m.OnInstallRuntime = func() { s.downloadNativeRelease(win, "", true) }
	m.OnInstallRelease = func(packageID string) { s.downloadNativeRelease(win, packageID, !m.VendorReady) }
	m.OnInstallUpdate = func() { s.downloadNativeRelease(win, "", !m.VendorReady) }
	m.OnSelectRelease = func(packageID string) {
		s.mu.Lock()
		if !s.nativeReleases.busy {
			s.nativeReleases.selected = packageID
		}
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}
	m.OnCancelRelease = func() {
		s.mu.Lock()
		cancel := s.nativeReleases.cancel
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	}
	m.OnStartStagedRelease = func() { s.startNativeStaged(win) }
	// The launcher notice must refer to the same pinned native catalog that the
	// install action uses, never the older untyped feed.
	m.UpdateAvailable = false
	m.UpdateStatus = ""
	for _, row := range snapshot.rows {
		if row.UnavailableReason == "" && (m.VendorReady || row.RuntimeIncluded) && !row.Prerelease && update.CompareVersions(row.Version, buildVersion) > 0 {
			m.UpdateAvailable = true
			m.UpdateStatus = fmt.Sprintf(i18n.T(i18n.KeyCmdMygoIsAvailable), row.Version)
			break
		}
	}
	s.maybeAskStartupUpdate(win, snapshot, m.VendorReady)
	s.mu.Lock()
	pendingUpdate := s.startupRecovery.updateAttention.Pending()
	s.mu.Unlock()
	if pendingUpdate && m.UpdateAvailable {
		m.UpdateStatus += i18n.T(i18n.KeyCmdAwaitingConfirmationChooseLaterOrDownl)
	}
	return s.withApplicationUninstall(win, s.withStartupDiagnostics(s.withWhaleBridgeAlerts(m)))
}

func (s *mainState) nativeReleaseNotify(win *mygo.Window) {
	if win != nil {
		win.Update(func() { s.refreshStartupUpdateAttention(win) })
	}
}

func (s *mainState) refreshNativeReleases(win *mygo.Window) {
	route := s.updateDownloadRoute()
	s.mu.Lock()
	if s.nativeReleases.busy {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.nativeReleases.attentionGeneration++
	s.nativeReleases.initialized = true
	s.nativeReleases.route = route
	s.nativeReleases.busy, s.nativeReleases.cancel = true, cancel
	s.nativeReleases.status = i18n.T(i18n.KeyCmdLoadingMygoReleaseCatalog)
	s.nativeReleases.phase = "checking"
	s.nativeReleases.err = ""
	s.nativeReleases.received, s.nativeReleases.total = 0, 0
	s.mu.Unlock()
	go func() {
		defer cancel()
		rows, err := update.ListNativeReleases(ctx, route)
		if err == nil {
			rows = update.PrepareNativeReleases(rows, buildVersion, update.NativeInstallerCacheDir(filepath.Join(paths.UserDataDir(), "updates")))
		}
		s.mu.Lock()
		r := &s.nativeReleases
		r.busy, r.cancel = false, nil
		if err != nil {
			r.err = err.Error()
			r.status = i18n.T(i18n.KeyCmdCatalogLookupFailedThePreviousListAnd)
			if errors.Is(err, context.Canceled) {
				r.status = i18n.T(i18n.KeyCmdReleaseLookupCancelled)
			}
		} else {
			r.rows = rows
			r.catalogRoute = route
			r.status = fmt.Sprintf(i18n.T(i18n.KeyCmdLoadedMygoReleases), len(rows))
			if len(rows) == 0 {
				r.status = i18n.T(i18n.KeyCmdNoCompatibleMygoInstallerHasBeenPublis)
			}
			found := false
			for _, row := range rows {
				if row.PackageKey() == r.selected && row.UnavailableReason == "" {
					found = true
					break
				}
			}
			if !found {
				r.selected = ""
				for _, row := range rows {
					if row.UnavailableReason == "" {
						r.selected = row.PackageKey()
						break
					}
				}
			}
		}
		r.phase = settledNativeReleasePhase(*r)
		s.mu.Unlock()
		s.nativeReleaseNotify(win)
	}()
	s.nativeReleaseNotify(win)
}

func (s *mainState) downloadNativeRelease(win *mygo.Window, tag string, requireRuntime bool, expected ...startuprec.UpdateAttentionScope) {
	route := s.updateDownloadRoute()
	s.mu.Lock()
	r := &s.nativeReleases
	if len(expected) > 0 {
		scope, _ := s.startupUpdateScopeLocked(win, *r, !requireRuntime)
		if scope != expected[0] || route != expected[0].Route || tag != expected[0].Tag || s.cfgStore == nil || !s.cfgStore.Get().AskOnUpdate {
			s.mu.Unlock()
			return
		}
		s.startupRecovery.updateHold = true
	}
	if r.busy {
		s.mu.Unlock()
		return
	}
	if requireRuntime && strings.TrimSpace(os.Getenv(paths.EnvVendor)) != "" {
		r.err = i18n.T(i18n.KeyCmdWhaleisleVendorOverridesTheRuntimePath)
		r.phase = "error"
		s.mu.Unlock()
		s.nativeReleaseNotify(win)
		return
	}
	rows := append([]update.NativeRelease(nil), r.rows...)
	if r.catalogRoute != route {
		rows = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.attentionGeneration++
	s.startupRecovery.updateAttention.DismissCurrent()
	r.initialized, r.route, r.busy, r.cancel = true, route, true, cancel
	r.status, r.err = i18n.T(i18n.KeyCmdSelectingAndVerifyingAMygoInstaller), ""
	r.phase = ""
	if len(rows) == 0 {
		r.phase = "checking"
	}
	r.received, r.total = 0, 0
	s.mu.Unlock()
	go func() {
		defer cancel()
		var selected update.NativeRelease
		var staged update.ApplyResult
		var staging string
		err := func() error {
			if len(rows) == 0 {
				var err error
				rows, err = update.ListNativeReleases(ctx, route)
				if err != nil {
					return err
				}
			}
			cacheDir := update.NativeInstallerCacheDir(filepath.Join(paths.UserDataDir(), "updates"))
			rows = update.PrepareNativeReleases(rows, buildVersion, cacheDir)
			var selectErr error
			selected, selectErr = update.SelectNativeRelease(rows, tag, runtime.GOOS, runtime.GOARCH, requireRuntime)
			if selectErr != nil {
				return selectErr
			}
			s.mu.Lock()
			s.nativeReleases.selected = selected.PackageKey()
			s.nativeReleases.status = i18n.T(i18n.KeyCmdDownloadingAndVerifyingFullInstaller) + selected.Tag
			if selected.IsDelta() {
				s.nativeReleases.status = i18n.T(i18n.KeyCmdReusingCachedInstallerAndDownloadingCh) + selected.Tag
			}
			s.mu.Unlock()
			// The launcher's native release download keeps the same evidence as
			// the in-app updater (update-journal.js): the request, integer
			// progress, then the verified result.
			s.recordUpdateAction(update.JournalActionDownloadRequested)
			startPercent := 0.0
			s.recordUpdateState(update.JournalState{Phase: update.JournalPhaseDownloading, Version: selected.Version, Percent: &startPercent})
			base := filepath.Join(paths.UserDataDir(), "updates")
			if err := os.MkdirAll(base, 0755); err != nil {
				return err
			}
			var err error
			staging, err = os.MkdirTemp(base, "mygo-native-")
			if err != nil {
				return err
			}
			staged, err = update.StageNativeReleaseWithOptions(ctx, selected, update.NativeStageOptions{CurrentVersion: buildVersion, CacheDir: cacheDir, RequireRuntime: requireRuntime, DestDir: staging, OnPhase: func(phase update.StagePhase) {
				s.mu.Lock()
				s.nativeReleases.phase = string(phase)
				s.mu.Unlock()
				s.nativeReleaseNotify(win)
			}, OnProgress: func(received, total int64) {
				s.mu.Lock()
				s.nativeReleases.received, s.nativeReleases.total = received, total
				s.mu.Unlock()
				percent := updateDownloadPercent(received, total)
				s.recordUpdateState(update.JournalState{Phase: update.JournalPhaseDownloading, Version: selected.Version, Percent: &percent})
				s.nativeReleaseNotify(win)
			}})
			if err != nil {
				return err
			}
			return ctx.Err()
		}()
		if err != nil && staging != "" {
			_ = os.RemoveAll(staging)
		}
		s.mu.Lock()
		r := &s.nativeReleases
		r.busy, r.cancel = false, nil
		if len(rows) > 0 {
			r.rows, r.catalogRoute = rows, route
		}
		var journal update.JournalState
		journal.Phase = update.JournalPhaseDownloaded
		journal.Version = selected.Version
		if err != nil {
			r.err = err.Error()
			r.status = i18n.T(i18n.KeyCmdInstallationPreparationFailedTheCurren)
			if errors.Is(err, context.Canceled) {
				r.status = i18n.T(i18n.KeyCmdDownloadCancelledTheCurrentInstallatio)
				r.err = ""
				journal = update.JournalState{}
			} else {
				journal = update.JournalState{Phase: update.JournalPhaseError, FailedOperation: "download", Message: err.Error()}
			}
		} else {
			r.staged, r.stagedTag = staged, selected.Tag
			variant := i18n.T(i18n.KeyCmdDesktopOnly)
			if selected.RuntimeIncluded {
				variant = i18n.T(i18n.KeyCmdIncludesRuntime)
			}
			kind := i18n.T(i18n.KeyCmdFullPackage)
			if selected.IsDelta() {
				kind = i18n.T(i18n.KeyCmdDeltaReconstruction)
			}
			r.stagedTitle = selected.Tag + " · " + kind + " · " + variant
			r.status = i18n.T(i18n.KeyCmdInstallerDownloadedAndVerifiedChooseIn)
			if selected.Platform == "darwin" {
				r.status = i18n.T(i18n.KeyCmdDiskImageDownloadedAndVerifiedChooseOp)
			}
			if selected.IsDelta() && !staged.UsedRange && staged.CopiedBlocks == 0 {
				r.status += i18n.T(i18n.KeyCmdTheSourceDidNotSupportRangesTheComple)
			}
			if strings.Contains(staged.Message, "native base cache unavailable:") {
				r.status += i18n.T(i18n.KeyCmdTheInstallerIsRetainedButCachingItFor)
			}
			r.err = ""
		}
		r.phase = settledNativeReleasePhase(*r)
		s.mu.Unlock()
		if journal.Phase != "" {
			// File evidence is written outside the state lock: it never delays a
			// painted frame that waits on this mutex.
			s.recordUpdateState(journal)
		}
		s.nativeReleaseNotify(win)
	}()
	s.nativeReleaseNotify(win)
}

func (s *mainState) startNativeStaged(win *mygo.Window) {
	s.mu.Lock()
	staged, busy := s.nativeReleases.staged, s.nativeReleases.busy
	s.mu.Unlock()
	if busy || staged.Path == "" {
		return
	}
	if !update.CanStartStaged(staged) {
		s.mu.Lock()
		s.nativeReleases.phase = "error"
		s.nativeReleases.err = i18n.T(i18n.KeyCmdThisPlatformCannotStartTheInstaller)
		s.mu.Unlock()
		s.nativeReleaseNotify(win)
		return
	}
	// Existing task admission, saved-session protection and last-moment hash
	// verification remain the only route that can launch the staged installer.
	s.mu.Lock()
	if s.nativeReleases.busy || s.taskRestartPending || s.quitting || s.launcherImportBusy {
		s.mu.Unlock()
		return
	}
	s.nativeReleases.busy = true
	s.nativeReleases.installing = true
	s.nativeReleases.phase = "installing"
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.nativeReleases.busy = false
			s.nativeReleases.installing = false
			s.nativeReleases.phase = settledNativeReleasePhase(s.nativeReleases)
			s.mu.Unlock()
			s.nativeReleaseNotify(win)
		}()
		if !s.installStagedUpdateWithConsent(win, staged, true) {
			s.mu.Lock()
			s.nativeReleases.status = i18n.T(i18n.KeyCmdVerifiedInstallerRetainedForRetryTheCu)
			s.mu.Unlock()
			s.nativeReleaseNotify(win)
		}
	}()
	s.nativeReleaseNotify(win)
}

func settledNativeReleasePhase(r nativeReleaseState) string {
	if r.err != "" {
		return "error"
	}
	if r.staged.Path != "" {
		return "ready"
	}
	for _, row := range r.rows {
		if row.UnavailableReason == "" && !row.Prerelease && update.CompareVersions(row.Version, buildVersion) > 0 {
			return "available"
		}
	}
	return ""
}
