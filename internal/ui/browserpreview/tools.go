package browserpreview

import (
	"context"
	"errors"
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
)

type browserTools struct {
	resize       *browserViewportDrag
	settings     browserToolSettings
	operation    *browserToolOperation
	confirmation *browserDataConfirmation[*desktop.Host]
}
type browserToolOperation struct {
	host   *desktop.Host
	epoch  uint64
	ctx    context.Context
	cancel context.CancelFunc
}

func (b *Controller) browserToolsCurrent(t *Tab, h *desktop.Host, epoch uint64) bool {
	return t != nil && b.current() == t && t.Document == nil && t.HTML == nil && t.Media == nil && t.Host == h && h != nil && h.Alive() && h.BrowserPageVersion() == epoch
}
func (b *Controller) cancelBrowserTools() {
	for _, t := range b.Tabs {
		b.cancelBrowserResize(t, true)
		t.tools.confirmation = nil
		if op := t.tools.operation; op != nil {
			op.cancel()
			t.tools.operation = nil
			t.ActionMessage = ""
		}
	}
}
func (b *Controller) pruneBrowserTools() {
	for _, t := range b.Tabs {
		if drag := t.tools.resize; drag != nil && !b.browserResizeCurrent(t, drag) {
			b.cancelBrowserResize(t, true)
		}
		if op := t.tools.operation; op != nil && (!b.browserToolsCurrent(t, op.host, op.epoch) || op.ctx.Err() != nil) {
			op.cancel()
			t.tools.operation = nil
			t.ActionMessage = ""
		}
		if confirm := t.tools.confirmation; confirm != nil && !b.browserToolsCurrent(t, confirm.host, confirm.epoch) {
			t.tools.confirmation = nil
		}
	}
}
func (b *Controller) startBrowserTool(t *Tab, kind, scheme string) {
	if t == nil || b.current() != t || t.Host == nil || t.tools.operation != nil || b.parent == nil {
		return
	}
	h, epoch := t.Host, t.Host.BrowserPageVersion()
	if kind != "scheme" && !isBrowserDataClear(kind) {
		return
	}
	if isBrowserDataClear(kind) && !takeBrowserDataConfirmation(&t.tools.confirmation, kind, h, epoch) {
		return
	}
	b.CancelAction()
	if !b.browserToolsCurrent(t, h, epoch) {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	op := &browserToolOperation{h, epoch, ctx, cancel}
	t.tools.operation = op
	t.Error = ""
	t.ActionMessage = browserChromeLabel("正在应用网页外观…", "Applying webpage appearance…")
	if kind == "cache" {
		t.ActionMessage = browserChromeLabel("正在清除当前分区「", "Clearing current profile “") + t.browserWorkspaceLabel() + browserChromeLabel("」的 HTTP 缓存…", "” HTTP cache…")
	} else if kind == "cookies" {
		t.ActionMessage = browserChromeLabel("正在清除当前分区「", "Clearing current profile “") + t.browserWorkspaceLabel() + browserChromeLabel("」的登录 Cookie…", "” sign-in cookies…")
	} else if kind == "site-data" {
		t.ActionMessage = browserChromeLabel("正在清除当前分区「", "Clearing current profile “") + t.browserWorkspaceLabel() + browserChromeLabel("」的五类站点数据…", "” five site-data categories…")
	}
	b.update()
	go func() {
		var err error
		if kind == "cache" {
			err = h.ClearBrowserHTTPCache(ctx)
		} else if kind == "cookies" {
			err = h.ClearBrowserLoginCookies(ctx, epoch)
		} else if kind == "site-data" {
			err = h.ClearBrowserSiteData(ctx, epoch)
		} else {
			err = h.SetBrowserColorScheme(ctx, mygo.BrowserColorScheme(scheme))
		}
		b.parent.Update(func() {
			if t.tools.operation != op {
				return
			}
			t.tools.operation = nil
			op.cancel()
			t.ActionMessage = ""
			if !b.browserToolsCurrent(t, h, epoch) {
				return
			}
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					t.Error = err.Error()
					if kind == "site-data" {
						t.Error = browserChromeLabel("五类站点数据清理未确认完成，可能已部分生效；不会自动重试：", "Completion of the five-category site-data clear was not confirmed; it may be partially applied and will not retry automatically: ") + err.Error()
					}
				}
				return
			}
			if kind == "cache" {
				t.ActionMessage = browserChromeLabel("当前分区 HTTP 缓存已清除；Cookies 和站点数据保留，其他工作区/插件不受影响", "Current profile HTTP cache cleared; cookies and site data retained. Other workspaces/plugins are unaffected.")
			} else if kind == "cookies" {
				t.ActionMessage = browserChromeLabel("当前分区全部 Cookie（含登录 Cookie）已清除；站点存储、Service Worker、HTTP 缓存保留，其他工作区/插件不受影响", "All cookies in this profile, including sign-in cookies, were cleared; site storage, service workers and HTTP cache retained. Other workspaces/plugins are unaffected.")
			} else if kind == "site-data" {
				t.ActionMessage = browserChromeLabel("当前分区五类站点数据已清除；HTTP 缓存、CacheStorage 等其他数据保留，其他工作区/插件不受影响。活跃站点可能再次创建数据。", "Five site-data categories cleared in this profile; HTTP cache, CacheStorage and other data retained. Other workspaces/plugins are unaffected. Active sites may recreate data.")
			} else {
				t.tools.settings.Scheme = scheme
				t.ActionMessage = browserChromeLabel("网页 prefers-color-scheme 已更新", "Webpage prefers-color-scheme updated")
			}
		})
	}()
}
func (b *Controller) updateBrowserDevice(t *Tab, w, h int, preset string) {
	if b.current() != t || !validBrowserDimensions(w, h) {
		return
	}
	b.CancelAction()
	t.tools.settings.setSize(w, h, preset)
	t.tools.settings.Device = true
	t.Error = ""
	if t.Host != nil {
		b.showHost(t.Host)
	}
	b.update()
}
func (b *Controller) changeBrowserZoom(t *Tab, zoom float64) {
	if b.current() != t || t.Host == nil {
		return
	}
	b.CancelAction()
	if err := t.Host.SetBrowserZoom(zoom); err != nil {
		t.Error = err.Error()
		return
	}
	t.Error = ""
	b.showHost(t.Host)
	b.update()
}

// browserDeviceViewport derives a fresh seat each frame; it never changes the
// original dock seat or applies settings from one tab to another Host.
func (b *Controller) browserDeviceViewport(h *desktop.Host) *mygo.Rectangle {
	t := b.current()
	if t == nil || t.Host != h || t.Document != nil || t.HTML != nil || t.Media != nil || !t.tools.settings.Device {
		return b.viewport
	}
	var seat mygo.Rectangle
	if b.viewport != nil {
		seat = *b.viewport
	} else {
		if b.parent == nil || b.parent.IsDestroyed() {
			return b.viewport
		}
		bounds := b.parent.ContentBounds()
		height := b.chromeHeight()
		seat = mygo.Rectangle{Y: chrome.TitleBarHeight + height, Width: bounds.Width, Height: max(0, bounds.Height-chrome.TitleBarHeight-height)}
	}
	s := t.tools.settings
	x, y, w, height, _ := browserDeviceLayout(seat.Width, seat.Height, s.Width, s.Height, h.BrowserZoom())
	return &mygo.Rectangle{X: seat.X + x, Y: seat.Y + y, Width: w, Height: height}
}

func (b *Controller) toolControls(c *ui.Context, t *Tab, state desktop.State) {
	if !browserWebTab(t) || !t.tools.settings.Device {
		return
	}
	s := &t.tools.settings
	disabled := t.Host == nil || state.Loading || t.tools.operation != nil || b.operation != nil || b.recording != nil
	ui.Row(c).FillWidth().MinWidth(0).Height(32).Gap(4).AlignItems(ui.Center).Label(browserPreviewText("deviceToolbarAria")).Children(func() {
		label := s.Preset
		if label == "" {
			label = browserPreviewText("viewportResponsive")
		}
		ui.MenuButton(c, label, func(menu *ui.Menu) {
			for _, preset := range browserDevicePresets {
				if menu.Item(fmt.Sprintf("%s · %d × %d", preset.Name, preset.Width, preset.Height)).Checked(s.Preset == preset.Name).Chosen() {
					b.updateBrowserDevice(t, preset.Width, preset.Height, preset.Name)
				}
			}
		}).Label(browserPreviewText("viewportPreset")).Height(28).MaxWidth(125).Disabled(disabled)
		submitted := ui.TextInput(c, &s.WidthDraft).Width(58).Height(28).Label(browserPreviewText("viewportWidth")).Disabled(disabled).Submitted()
		ui.Text(c, "×").FontSize(11)
		submitted = ui.TextInput(c, &s.HeightDraft).Width(58).Height(28).Label(browserPreviewText("viewportHeight")).Disabled(disabled).Submitted() || submitted
		if ui.Button(c, "↵").Size(28, 28).Label(browserPreviewText("viewportDimensions")).Disabled(disabled).Clicked() || submitted {
			w, h, err := s.dimensions(s.WidthDraft, s.HeightDraft)
			if err != nil {
				t.Error = err.Error()
			} else {
				b.updateBrowserDevice(t, w, h, "")
			}
		}
		if ui.Button(c, "↻").Size(28, 28).Label(browserPreviewText("viewportRotate")).Disabled(disabled).Clicked() {
			b.updateBrowserDevice(t, s.Height, s.Width, s.Preset)
		}
		ui.Box(c).Grow(1)
		if ui.Button(c, "×").Size(28, 28).Label(browserPreviewText("deviceToolbarHide")).Clicked() {
			s.Device = false
			b.showHost(t.Host)
			b.update()
		}
	})
}
