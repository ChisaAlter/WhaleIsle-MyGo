package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
			m.UpdateStatus = fmt.Sprintf(resumeText("可安装 MyGo %s", "MyGo %s is available"), row.Version)
			break
		}
	}
	s.maybeAskStartupUpdate(win, snapshot, m.VendorReady)
	s.mu.Lock()
	pendingUpdate := s.startupRecovery.updateAttention.Pending()
	s.mu.Unlock()
	if pendingUpdate && m.UpdateAvailable {
		m.UpdateStatus += resumeText("，等待确认。可在启动器中选择稍后或下载。", "; awaiting confirmation. Choose Later or Download in the launcher.")
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
	s.nativeReleases.status = resumeText("正在读取 MyGo 发布目录…", "Loading MyGo release catalog…")
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
			r.status = resumeText("目录读取失败；已保留此前列表和下载。", "Catalog lookup failed; the previous list and download were kept.")
			if errors.Is(err, context.Canceled) {
				r.status = resumeText("已取消读取发布目录。", "Release lookup cancelled.")
			}
		} else {
			r.rows = rows
			r.catalogRoute = route
			r.status = fmt.Sprintf(resumeText("已读取 %d 个 MyGo 发布条目。", "Loaded %d MyGo releases."), len(rows))
			if len(rows) == 0 {
				r.status = resumeText("尚未发布兼容的 MyGo 安装包。", "No compatible MyGo installer has been published.")
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
		r.err = resumeText("当前设置了 WHALEISLE_VENDOR；原生安装器不会覆盖自定义运行时目录。请修复该目录或取消此覆盖后重试。", "WHALEISLE_VENDOR overrides the runtime path. The native installer cannot replace that custom directory; repair it or remove the override before retrying.")
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
	r.status, r.err = resumeText("正在选择并校验 MyGo 安装包…", "Selecting and verifying a MyGo installer…"), ""
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
			s.nativeReleases.status = resumeText("正在下载并校验完整安装包：", "Downloading and verifying full installer: ") + selected.Tag
			if selected.IsDelta() {
				s.nativeReleases.status = resumeText("正在复用已缓存安装包并下载变化区块：", "Reusing cached installer and downloading changed blocks: ") + selected.Tag
			}
			s.mu.Unlock()
			base := filepath.Join(paths.UserDataDir(), "updates")
			if err := os.MkdirAll(base, 0755); err != nil {
				return err
			}
			var err error
			staging, err = os.MkdirTemp(base, "mygo-native-")
			if err != nil {
				return err
			}
			staged, err = update.StageNativeReleaseWithOptions(ctx, selected, update.NativeStageOptions{CurrentVersion: buildVersion, CacheDir: cacheDir, RequireRuntime: requireRuntime, DestDir: staging, OnProgress: func(received, total int64) {
				s.mu.Lock()
				s.nativeReleases.received, s.nativeReleases.total = received, total
				s.mu.Unlock()
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
		if err != nil {
			r.err = err.Error()
			r.status = resumeText("安装准备失败；当前安装、选择和此前已校验下载均保留。", "Installation preparation failed; the current installation, selection and previous verified download were kept.")
			if errors.Is(err, context.Canceled) {
				r.status = resumeText("已取消下载，未更改当前安装。", "Download cancelled; the current installation was not changed.")
				r.err = ""
			}
		} else {
			r.staged, r.stagedTag = staged, selected.Tag
			variant := resumeText("仅桌面端", "Desktop only")
			if selected.RuntimeIncluded {
				variant = resumeText("包含运行时", "Includes runtime")
			}
			kind := resumeText("完整包", "Full package")
			if selected.IsDelta() {
				kind = resumeText("差量重建", "Delta reconstruction")
			}
			r.stagedTitle = selected.Tag + " · " + kind + " · " + variant
			r.status = resumeText("安装包已下载并校验。点击“安装并退出”继续；仅安装所选发布明确列出的内容。", "Installer downloaded and verified. Choose Install and quit to install the contents declared by the selected release.")
			if selected.Platform == "darwin" {
				r.status = resumeText("磁盘映像已下载并校验。点击“打开磁盘映像并退出”，再将 Whale Isle.app 拖到 Applications 完成安装。", "Disk image downloaded and verified. Choose Open disk image and quit, then drag Whale Isle.app to Applications to finish installation.")
			}
			if selected.IsDelta() && !staged.UsedRange && staged.CopiedBlocks == 0 {
				r.status += resumeText(" 下载源未提供范围请求，已回退完整下载并校验。", " The source did not support ranges; the complete installer was downloaded and verified.")
			}
			if strings.Contains(staged.Message, "native base cache unavailable:") {
				r.status += resumeText(" 安装包已保留，但未能缓存后续差量更新基包；后续仍可使用完整包。", " The installer is retained, but caching it for future deltas failed; full packages remain available.")
			}
			r.err = ""
		}
		s.mu.Unlock()
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
		s.nativeReleases.err = resumeText("当前平台无法启动此安装器。", "This platform cannot start the installer.")
		s.mu.Unlock()
		s.nativeReleaseNotify(win)
		return
	}
	// Existing task admission, saved-session protection and last-moment hash
	// verification remain the only route that can launch the staged installer.
	if !s.installStagedUpdate(win, staged) {
		s.mu.Lock()
		s.nativeReleases.status = resumeText("已保留校验后的安装包，可稍后重试安装；当前窗口未关闭。", "Verified installer retained for retry; the current window remains open.")
		s.mu.Unlock()
		s.nativeReleaseNotify(win)
	}
}

