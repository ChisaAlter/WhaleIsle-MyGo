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
	"whaleisle/internal/ui/launcher"
)

type nativeReleaseState struct {
	initialized     bool
	route           string
	catalogRoute    string
	rows            []update.NativeRelease
	selected        string
	busy            bool
	cancel          context.CancelFunc
	status          string
	err             string
	received, total int64
	staged          update.ApplyResult
	stagedTag       string
}

func (s *mainState) withNativeReleases(win *mygo.Window, m launcher.Model) launcher.Model {
	route := s.updateDownloadRoute()
	s.mu.Lock()
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
	m.ReleaseSelectedTag = snapshot.selected
	m.ReleaseReceived, m.ReleaseTotal = snapshot.received, snapshot.total
	m.ReleaseStagedTag = snapshot.stagedTag
	m.OnRefreshReleases = func() { s.refreshNativeReleases(win) }
	m.OnCheckUpdate = m.OnRefreshReleases
	m.OnInstallRuntime = func() { s.downloadNativeRelease(win, "", true) }
	m.OnInstallRelease = func(tag string) { s.downloadNativeRelease(win, tag, !m.VendorReady) }
	m.OnInstallUpdate = func() { s.downloadNativeRelease(win, "", !m.VendorReady) }
	m.OnSelectRelease = func(tag string) {
		s.mu.Lock()
		if !s.nativeReleases.busy {
			s.nativeReleases.selected = tag
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
	return withApplicationUninstall(win, s.withStartupDiagnostics(m))
}

func (s *mainState) nativeReleaseNotify(win *mygo.Window) {
	if win != nil {
		win.Update(func() {})
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
				if row.Tag == r.selected && row.UnavailableReason == "" {
					found = true
					break
				}
			}
			if !found {
				r.selected = ""
				for _, row := range rows {
					if row.UnavailableReason == "" {
						r.selected = row.Tag
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

func (s *mainState) downloadNativeRelease(win *mygo.Window, tag string, requireRuntime bool) {
	route := s.updateDownloadRoute()
	s.mu.Lock()
	r := &s.nativeReleases
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
			for _, candidate := range rows {
				if tag != "" && candidate.Tag != tag {
					continue
				}
				if tag == "" && candidate.Prerelease {
					continue
				}
				if candidate.Validate(runtime.GOOS, runtime.GOARCH, requireRuntime) == nil {
					selected = candidate
					break
				}
			}
			if selected.Tag == "" {
				return fmt.Errorf("no compatible MyGo release for this selection; first installation requires a native package with matching Host and Node runtime")
			}
			s.mu.Lock()
			s.nativeReleases.selected = selected.Tag
			s.nativeReleases.status = resumeText("正在下载并校验：", "Downloading and verifying: ") + selected.Tag
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
			staged, err = update.StageNativeRelease(ctx, selected, requireRuntime, staging, func(received, total int64) {
				s.mu.Lock()
				s.nativeReleases.received, s.nativeReleases.total = received, total
				s.mu.Unlock()
				s.nativeReleaseNotify(win)
			})
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
			r.status = resumeText("安装包已下载并校验。点击“安装并退出”继续；仅安装所选发布明确列出的内容。", "Installer downloaded and verified. Choose Install and quit to install the contents declared by the selected release.")
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
