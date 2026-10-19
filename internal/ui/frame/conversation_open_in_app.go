package frame

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"sync"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

type workspaceApplication struct {
	id   string
	icon *ui.Bitmap
}

// WorkspaceOpenInAppState has the original controller's lifetime: availability
// and images survive Session/page changes; the captured Host owns their carrier.
type WorkspaceOpenInAppState struct {
	mu           sync.Mutex
	source       *session.OpenInAppSource
	key          string
	ctx          context.Context
	cancel       context.CancelFunc
	closed, busy bool
	apps         []workspaceApplication
	choice       string
	save         func(string) error
	notify       func()
	rendered     bool
	viewPath     string
	viewEpoch    uint64
	failure      string
	toastID      string
	retiredToast string
}

func NewWorkspaceOpenInAppState(choice string, save func(string) error) *WorkspaceOpenInAppState {
	return &WorkspaceOpenInAppState{choice: choice, save: save}
}

// ObserveHost runs even while the header is absent. A failed catalog read stays
// empty for this Host, matching the once-per-controller source load contract.
func (s *WorkspaceOpenInAppState) ObserveHost(store *session.Store) {
	if s == nil {
		return
	}
	var source *session.OpenInAppSource
	if store != nil {
		source = store.OpenInAppSource()
	}
	key := ""
	if source != nil {
		key = source.Key()
	}
	s.mu.Lock()
	if s.closed || key == s.key {
		s.mu.Unlock()
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.source, s.key, s.apps, s.busy = source, key, nil, false
	s.retireViewLocked()
	if source == nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx, s.cancel = ctx, cancel
	s.mu.Unlock()
	go func() {
		ids, _ := source.Applications(ctx)
		apps := make([]workspaceApplication, 0, len(ids))
		seen := make(map[string]bool)
		for _, id := range ids {
			if _, known := workspaceApplicationNames[id]; known && !seen[id] {
				apps = append(apps, workspaceApplication{id: id})
				seen[id] = true
			}
		}
		s.mu.Lock()
		if !s.currentLocked(source, key, ctx) {
			s.mu.Unlock()
			return
		}
		s.apps = apps
		notify := s.notify
		s.mu.Unlock()
		if notify != nil {
			notify()
		}
		for _, app := range apps {
			app := app
			go s.loadIcon(source, key, ctx, app.id)
		}
	}()
}

func (s *WorkspaceOpenInAppState) currentLocked(source *session.OpenInAppSource, key string, ctx context.Context) bool {
	return !s.closed && s.source == source && s.key == key && ctx.Err() == nil && source.Current()
}

func (s *WorkspaceOpenInAppState) loadIcon(source *session.OpenInAppSource, key string, ctx context.Context, id string) {
	data, err := source.Icon(ctx, id)
	if err != nil {
		return
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width) > imagepreview.MaxPixels/int64(config.Height) {
		return
	}
	bitmap, err := ui.DecodeBitmap(data)
	if err != nil {
		return
	}
	s.mu.Lock()
	if !s.currentLocked(source, key, ctx) {
		s.mu.Unlock()
		return
	}
	for i := range s.apps {
		if s.apps[i].id == id {
			s.apps[i].icon = bitmap
			break
		}
	}
	notify := s.notify
	s.mu.Unlock()
	if notify != nil {
		notify()
	}
}

func (s *WorkspaceOpenInAppState) retireViewLocked() {
	s.viewEpoch++
	s.viewPath, s.failure = "", ""
	if s.toastID != "" {
		s.retiredToast, s.toastID = s.toastID, ""
	}
}

// FinishFrame only retires feedback owned by an unmounted initiating control.
// It deliberately keeps the controller's application and icon cache.
func (s *WorkspaceOpenInAppState) FinishFrame(c *ui.Context) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.rendered && s.viewPath != "" {
		s.retireViewLocked()
	}
	s.rendered = false
	toast := s.retiredToast
	s.retiredToast = ""
	s.mu.Unlock()
	if toast != "" {
		c.CloseToast(toast)
	}
}

func (s *WorkspaceOpenInAppState) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.cancel != nil {
		s.cancel()
	}
	s.apps, s.source, s.busy = nil, nil, false
	s.retireViewLocked()
}

// OpenCurrent is the existing workspace.openLocal command's action. Its caller
// retains the original main-conversation/no-active-panel command scope.
func (s *WorkspaceOpenInAppState) OpenCurrent(store *session.Store) bool {
	if s == nil || store == nil {
		return false
	}
	s.ObserveHost(store)
	snap := store.Snapshot()
	s.mu.Lock()
	id := ""
	if len(s.apps) > 0 {
		id = s.apps[0].id
	}
	for _, app := range s.apps {
		if app.id == s.choice {
			id = app.id
			break
		}
	}
	s.mu.Unlock()
	if id == "" {
		return false
	}
	return s.launch(snap.SelectedID, snap.SelectedCwd, id, false)
}

func (s *WorkspaceOpenInAppState) launch(sessionID, path, app string, remember bool) bool {
	s.mu.Lock()
	if s.closed || s.busy || s.source == nil {
		s.mu.Unlock()
		return false
	}
	installed := false
	for _, entry := range s.apps {
		if entry.id == app {
			installed = true
			break
		}
	}
	if !installed {
		s.mu.Unlock()
		return false
	}
	source, key, ctx, epoch := s.source, s.key, s.ctx, s.viewEpoch
	open, captureErr := source.CaptureOpen(sessionID, path, app)
	if captureErr != nil {
		s.mu.Unlock()
		return false
	}
	s.busy = true
	notify := s.notify
	s.mu.Unlock()
	if notify != nil {
		notify()
	}
	go func() {
		err := open(ctx)
		s.mu.Lock()
		if !s.currentLocked(source, key, ctx) {
			s.mu.Unlock()
			return
		}
		s.busy = false
		failure := ""
		if err != nil {
			failure = archiveText("打开失败，请重试", "Could not open. Try again.")
		} else if remember {
			s.choice = app
			if s.save != nil && s.save(app) != nil {
				failure = archiveText("已打开应用，但无法记住本次选择。", "Application opened, but the choice could not be saved.")
			}
		}
		if s.viewEpoch == epoch && s.viewPath == path {
			s.failure = failure
		}
		notify := s.notify
		s.mu.Unlock()
		if notify != nil {
			notify()
		}
	}()
	return true
}

func conversationOpenInApp(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors) {
	s := m.WorkspaceOpenInApp
	if s == nil || snap.SelectedID == "" || snap.SelectedCwd == "" {
		return
	}
	s.ObserveHost(m.Store)
	s.mu.Lock()
	s.notify = c.Services().Invalidate
	if s.closed || len(s.apps) == 0 {
		s.mu.Unlock()
		return
	}
	if s.viewPath != snap.SelectedCwd {
		s.retireViewLocked()
		s.viewPath = snap.SelectedCwd
	}
	s.rendered = true
	apps := append([]workspaceApplication(nil), s.apps...)
	preferred := apps[0]
	for _, app := range apps {
		if app.id == s.choice {
			preferred = app
			break
		}
	}
	busy, failure := s.busy, s.failure
	s.failure = ""
	if failure != "" {
		s.toastID = fmt.Sprintf("workspace-open-in-app:%p:%d", s, s.viewEpoch)
	}
	toastID := s.toastID
	s.mu.Unlock()
	if failure != "" {
		c.AddToast(ui.Toast{ID: toastID, Title: failure, Type: "error", Timeout: -1})
	}
	ink, border := chrome.Color(colors.FG), chrome.Color(colors.Border)
	root := ui.Row(c.Key("workspace-open-in-app:"+snap.SelectedCwd)).Height(27).Shrink(0).Radius(14).Clip().Border(.5, border).AlignItems(ui.Center)
	root.Children(func() {
		label := fmt.Sprintf(archiveText("用 %s 打开", "Open in %s"), workspaceApplicationName(preferred.id))
		button := ui.ButtonBase(c.Key("workspace-open-in-app-primary")).Label(label).Tooltip(label).Height(26).Padding(0, 9).BorderWidth(0).Background(ui.Transparent).Disabled(busy)
		if !busy && (button.Hovered() || button.FocusVisible()) {
			button.Background(chrome.Color(colors.Hover))
		}
		button.Children(func() {
			if busy {
				ui.Box(c).Size(13, 13)
			} else {
				workspaceApplicationIcon(c, preferred, 13, ink)
			}
		})
		// Disabled toolkit elements dim their descendants. The original pending
		// pill keeps full opacity; its visual seat lives beside the disabled hit
		// target while busy, so keyboard/pointer admission remains disabled.
		if busy {
			ui.Box(c).Absolute().AttachTo(button, ui.AnchorCenter, ui.AnchorCenter).Size(13, 13).Children(func() { ui.Spinner(c).Size(13, 13).TextColor(ink) })
		}
		if button.Clicked() {
			s.launch(snap.SelectedID, snap.SelectedCwd, preferred.id, false)
		}
		if len(apps) <= 1 {
			return
		}
		ui.Box(c).Width(.5).Height(26).Background(border)
		choice := ui.Local(root, "menu-choice", func() string { return "" })
		wasOpen := ui.Local(root, "menu-open", func() bool { return false })
		parts := ui.SelectBase(c.Key("workspace-open-in-app-menu"), choice)
		parts.Trigger.Label(archiveText("更多打开方式", "More ways to open")).Tooltip(archiveText("更多打开方式", "More ways to open")).Size(24, 26).Padding(0).BorderWidth(0).Background(ui.Transparent).Disabled(busy).Children(func() {
			if busy {
				ui.Box(c).Size(14, 14)
			} else {
				ui.Icon(c, workspaceOpenChevron).Size(14, 14).TextColor(ink)
			}
		})
		if !busy && (parts.Trigger.Hovered() || parts.Trigger.FocusVisible()) {
			parts.Trigger.Background(chrome.Color(colors.Hover))
		}
		if busy {
			ui.Box(c).Absolute().AttachTo(parts.Trigger, ui.AnchorCenter, ui.AnchorCenter).Size(14, 14).Children(func() { ui.Icon(c, workspaceOpenChevron).Size(14, 14).TextColor(ink) })
		}
		if !busy {
			parts.Popup(func(panel ui.Element) {
				panel.AttachTo(parts.Trigger, ui.AnchorBottomRight, ui.AnchorTopRight).MinWidth(180).MaxHeight(400).Padding(4).Radius(6).Border(.5, border).Background(chrome.Color(colors.BG))
				ui.Scroll(c).FillWidth().MaxHeight(360).MinHeight(0).Children(func() {
					for _, app := range apps {
						app := app
						name := workspaceApplicationName(app.id)
						if app.id == preferred.id {
							name = fmt.Sprintf(archiveText("%s（默认）", "%s (default)"), name)
						}
						item := parts.Item(app.id).Label(name).FillWidth().Padding(6, 8).Radius(4)
						if item.Highlighted() {
							item.Background(chrome.Color(colors.Hover)).ScrollIntoView()
						}
						item.Children(func() {
							ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
								workspaceApplicationIcon(c, app, 14, ink)
								ui.Text(c, name).FontSize(theme.InterfaceSize(12)).TextColor(ink).SingleLine()
							})
						})
					}
				})
			})
		}
		if parts.Open() && !busy && !*wasOpen {
			parts.Highlight(apps[0].id)
		}
		*wasOpen = parts.Open() && !busy
		if parts.Trigger.Changed() {
			id := *choice
			*choice = ""
			s.launch(snap.SelectedID, snap.SelectedCwd, id, true)
		}
	})
}

func workspaceApplicationIcon(c *ui.Context, app workspaceApplication, size float32, ink ui.Color) {
	if app.icon != nil {
		ui.Image(c, app.icon).Size(size, size).Fit(ui.Contain)
	} else {
		ui.Icon(c, workspaceOpenFallback).Size(size, size).TextColor(ink)
	}
}

func workspaceApplicationName(id string) string {
	switch id {
	case "finder":
		return archiveText("访达", "Finder")
	case "explorer":
		return archiveText("文件资源管理器", "File Explorer")
	case "filemanager":
		return archiveText("文件管理器", "Files")
	case "terminal":
		return archiveText("终端", "Terminal")
	}
	return workspaceApplicationNames[id]
}

var workspaceApplicationNames = map[string]string{
	"finder": "Finder", "explorer": "File Explorer", "filemanager": "Files", "cursor": "Cursor", "vscode": "VS Code", "vscodeinsiders": "VS Code Insiders", "windsurf": "Windsurf", "zed": "Zed", "sublimetext": "Sublime Text", "xcode": "Xcode", "androidstudio": "Android Studio", "intellij": "IntelliJ IDEA", "pycharm": "PyCharm", "webstorm": "WebStorm", "phpstorm": "PhpStorm", "goland": "GoLand", "rider": "Rider", "rustrover": "RustRover", "fork": "Fork", "sourcetree": "Sourcetree", "github": "GitHub Desktop", "tower": "Tower", "gitkraken": "GitKraken", "smartgit": "SmartGit", "sublimemerge": "Sublime Merge", "ghostty": "Ghostty", "warp": "Warp", "iterm": "iTerm2", "kitty": "kitty", "terminal": "Terminal", "windowsterminal": "Windows Terminal", "gitbash": "Git Bash", "gnometerminal": "GNOME Terminal", "konsole": "Konsole",
}

var workspaceOpenChevron = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M4 6L7.29289 9.29289C7.68342 9.68342 8.31658 9.68342 8.70711 9.29289L12 6" stroke="currentColor"/></svg>`))
var workspaceOpenFallback = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M11.7256 2.77441C12.5538 2.77469 13.2256 3.44616 13.2256 4.27441V10.1416H12.2256V4.27441C12.2256 3.99844 12.0015 3.77469 11.7256 3.77441H5.7207V2.77441H11.7256Z" fill="currentColor"/><path d="M2.77441 13.2255L12.3756 3.62427" stroke="currentColor"/></svg>`))
