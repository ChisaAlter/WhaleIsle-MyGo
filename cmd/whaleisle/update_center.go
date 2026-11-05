package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
)

type nativeUpdateRecord struct {
	Version                              int
	Route, Phase, StagedTag, StagedTitle string
	AutoDownload                         bool
	RequiredRuntime                      bool
	ExplicitVersion, InstallRequested    bool
	Received, Total                      int64
	Active                               update.NativeRelease
	Staged                               update.ApplyResult
}

var nativeUpdateDiskMu sync.Mutex

func nativeReleaseStagingDir(r update.NativeRelease) string {
	return filepath.Join(paths.UserDataDir(), "updates", "native-partials", url.PathEscape(r.PackageKey()))
}

func (s *mainState) ensureNativeUpdateCenter() {
	nativeUpdateDiskMu.Lock()
	defer nativeUpdateDiskMu.Unlock()
	s.mu.Lock()
	if s.nativeReleases.centerLoaded {
		s.mu.Unlock()
		return
	}
	s.nativeReleases.centerLoaded = true
	cfg := s.cfgStore
	s.nativeReleases.autoDownload = true
	if cfg != nil {
		s.nativeReleases.autoDownload = cfg.Get().AutoDownloadUpdates
	}
	s.mu.Unlock()
	name := filepath.Join(paths.UserDataDir(), "updates", "update-center.json")
	raw, err := os.ReadFile(name)
	if os.IsNotExist(err) {
		return
	}
	var record nativeUpdateRecord
	if err == nil {
		err = json.Unmarshal(raw, &record)
		if err == nil && record.Version != 1 {
			err = fmt.Errorf("unsupported update-center version")
		}
	}
	if err != nil {
		preserved := name + ".invalid-" + time.Now().UTC().Format("20060102T150405")
		if os.Rename(name, preserved) != nil {
			preserved = name
		}
		s.mu.Lock()
		s.nativeReleases.err = lifecycleText("更新恢复状态无法读取，原文件保留于 ", "Could not read update recovery state; original retained at ") + preserved
		s.nativeReleases.phase = "error"
		s.mu.Unlock()
		return
	}
	// Older native centers stored the preference in their journal. Migrate it
	// only when the shared desktop config has no explicit value; afterwards the
	// original autoDownloadUpdates field remains the single preference owner.
	if cfg != nil && !cfg.HasAutoDownloadUpdates() {
		if err := cfg.Update(func(c *config.Config) { c.AutoDownloadUpdates = record.AutoDownload }); err != nil {
			s.mu.Lock()
			s.nativeReleases.err = lifecycleText("无法迁移自动下载偏好：", "Could not migrate automatic download preference: ") + err.Error()
			s.mu.Unlock()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &s.nativeReleases
	if cfg != nil {
		r.autoDownload = cfg.Get().AutoDownloadUpdates
	} else {
		r.autoDownload = record.AutoDownload
	}
	// Explicit version selection may intentionally stage a reinstall/downgrade.
	// Once installation was requested it must reconcile against the installed
	// version, just like an automatic update; failed newer installs stay ready.
	if record.Active.Tag != "" && update.CompareVersions(record.Active.Version, buildVersion) <= 0 && (!record.ExplicitVersion || record.InstallRequested) {
		r.status = lifecycleText("已完成的更新记录已退役。", "The completed update record was retired.")
		if err := writeNativeUpdateRecord(nativeUpdateRecord{Version: 1, AutoDownload: r.autoDownload}); err != nil {
			r.err = lifecycleText("保存更新状态失败：", "Could not save update state: ") + err.Error()
		}
		return
	}
	if record.Active.Tag != "" && record.Active.Validate(runtime.GOOS, runtime.GOARCH, false) == nil {
		r.active = record.Active
		r.active.DownloadRoute = record.Route
		r.route = record.Route
		r.rows = []update.NativeRelease{r.active}
		r.catalogRoute = record.Route
		r.initialized = true
		r.requireRuntime = record.RequiredRuntime
		r.explicitVersion, r.installRequested = record.ExplicitVersion, record.InstallRequested
		r.selected = r.active.PackageKey()
		r.staging = nativeReleaseStagingDir(r.active)
		r.received, r.total = record.Received, record.Total
		if (record.Phase == "ready" || record.Phase == "installing") && record.Staged.Path != "" {
			base := filepath.Clean(r.staging) + string(filepath.Separator)
			if rel, err := filepath.Rel(base, record.Staged.Path); err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				if st, err := os.Stat(record.Staged.Path); err == nil && st.Mode().IsRegular() && st.Size() == record.Active.Size && record.Staged.Size == record.Active.Size && strings.EqualFold(record.Staged.SHA256, record.Active.SHA256) {
					r.staged = record.Staged
					r.stagedTag = record.Active.Tag
					r.stagedTitle = record.StagedTitle
					r.phase = "ready"
					r.status = lifecycleText("已恢复安装包记录，安装前将再次校验。", "Installer record restored; it will be verified again before installation.")
					return
				}
			}
		}
		r.paused = true
		r.phase = "paused"
		r.status = lifecycleText("上次下载已暂停，继续将复用已下载内容。", "Previous download paused; resume reuses downloaded content.")
	}
}

func (s *mainState) persistNativeUpdateCenter() {
	nativeUpdateDiskMu.Lock()
	defer nativeUpdateDiskMu.Unlock()
	s.mu.Lock()
	r := s.nativeReleases
	route := r.route
	if r.active.DownloadRoute != "" {
		route = r.active.DownloadRoute
	}
	record := nativeUpdateRecord{Version: 1, Route: route, Phase: r.phase, AutoDownload: r.autoDownload, RequiredRuntime: r.requireRuntime, ExplicitVersion: r.explicitVersion, InstallRequested: r.installRequested, Active: r.active, Received: r.received, Total: r.total, Staged: r.staged, StagedTag: r.stagedTag, StagedTitle: r.stagedTitle}
	s.mu.Unlock()
	if err := writeNativeUpdateRecord(record); err != nil {
		s.mu.Lock()
		s.nativeReleases.err = lifecycleText("保存更新状态失败：", "Could not save update state: ") + err.Error()
		s.mu.Unlock()
	}
}

// Caller owns nativeUpdateDiskMu; all publication errors keep the previous
// journal and reach the product's existing error surface.
func writeNativeUpdateRecord(record nativeUpdateRecord) error {
	base := filepath.Join(paths.UserDataDir(), "updates")
	if err := os.MkdirAll(base, 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(base, "update-center-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(base, "update-center.json"))
}

func (s *mainState) openNativeUpdateCenter(win *mygo.Window) {
	s.ensureNativeUpdateCenter()
	mygo.RunOnMain(func() {
		owner := s.lifecycleDialogOwner(win)
		if owner == nil {
			s.mu.Lock()
			owner = s.win
			if s.launcherWin != nil && s.launcherWin.win != nil {
				owner = s.launcherWin.win
			}
			s.mu.Unlock()
			if owner == nil || owner.IsDestroyed() {
				return
			}
			owner.Restore()
			owner.Show()
			owner.Focus()
		}
		s.showNativeUpdateModal(owner)
	})
}
func (s *mainState) pauseNativeUpdate(win *mygo.Window) {
	s.mu.Lock()
	r := &s.nativeReleases
	if r.phase != "downloading" || !r.busy {
		s.mu.Unlock()
		return
	}
	r.pauseRequested = true
	cancel := r.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.nativeReleaseNotify(win)
}
func (s *mainState) resumeNativeUpdate(win *mygo.Window) {
	s.ensureNativeUpdateCenter()
	s.mu.Lock()
	r := &s.nativeReleases
	if r.busy || r.active.Tag == "" {
		s.mu.Unlock()
		return
	}
	r.paused = true
	key, requireRuntime := r.active.PackageKey(), r.requireRuntime
	s.mu.Unlock()
	s.downloadNativeRelease(win, key, requireRuntime)
}
func (s *mainState) cancelNativeUpdate(win *mygo.Window) {
	s.mu.Lock()
	r := &s.nativeReleases
	if r.busy {
		r.cancelRequested = true
		r.pauseRequested = false
		cancel := r.cancel
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return
	}
	dir := r.staging
	r.active = update.NativeRelease{}
	r.staged, r.stagedTag, r.stagedTitle = update.ApplyResult{}, "", ""
	r.staging = ""
	r.paused = false
	r.received, r.total = 0, 0
	r.err = ""
	r.status = lifecycleText("下载已取消。", "Download cancelled.")
	r.phase = settledNativeReleasePhase(*r)
	s.mu.Unlock()
	if dir != "" {
		if err := discardNativePartial(dir); err != nil {
			s.mu.Lock()
			s.nativeReleases.err = err.Error()
			s.nativeReleases.phase = "error"
			s.mu.Unlock()
		}
	}
	s.persistNativeUpdateCenter()
	s.nativeReleaseNotify(win)
}

func discardNativePartial(dir string) error {
	base, err := filepath.Abs(filepath.Join(paths.UserDataDir(), "updates", "native-partials"))
	if err != nil {
		return err
	}
	target, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to discard a path outside native update partials")
	}
	return os.RemoveAll(target)
}
func (s *mainState) setNativeAutoDownload(win *mygo.Window, enabled bool) {
	s.ensureNativeUpdateCenter()
	s.mu.Lock()
	cfg := s.cfgStore
	s.mu.Unlock()
	if cfg != nil {
		if err := cfg.Update(func(c *config.Config) { c.AutoDownloadUpdates = enabled }); err != nil {
			s.mu.Lock()
			s.nativeReleases.err = lifecycleText("保存自动下载偏好失败：", "Could not save automatic download preference: ") + err.Error()
			s.mu.Unlock()
			s.nativeReleaseNotify(win)
			return
		}
	}
	s.mu.Lock()
	s.nativeReleases.autoDownload = enabled
	s.mu.Unlock()
	s.persistNativeUpdateCenter()
	if enabled {
		s.maybeAutoDownloadNative(win)
	}
	s.nativeReleaseNotify(win)
}
func (s *mainState) maybeAutoDownloadNative(win *mygo.Window) {
	s.mu.Lock()
	r := s.nativeReleases
	requireRuntime := !s.startupRecovery.updateVendorReady
	allowed := r.autoDownload && !r.busy && !r.paused && r.err == "" && r.catalogRoute == r.route && r.staged.Path == "" && !s.quitting && !s.launcherImportBusy && !s.launcherStartBusy && !s.taskRestartPending
	var key string
	if allowed {
		for _, release := range r.rows {
			if !release.Prerelease && release.UnavailableReason == "" && (!requireRuntime || release.RuntimeIncluded) && update.CompareVersions(release.Version, buildVersion) > 0 {
				key = release.PackageKey()
				break
			}
		}
	}
	s.mu.Unlock()
	if key != "" {
		s.downloadNativeRelease(win, key, requireRuntime)
	}
}
