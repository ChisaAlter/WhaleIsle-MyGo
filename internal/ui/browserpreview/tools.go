package browserpreview

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	fonttheme "whaleisle/internal/ui/theme"

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
		seat = mygo.Rectangle{Y: chrome.TitleBarHeight + BrowserChromeHeight, Width: bounds.Width, Height: max(0, bounds.Height-chrome.TitleBarHeight-BrowserChromeHeight)}
	}
	s := t.tools.settings
	x, y, w, height, _ := browserDeviceLayout(seat.Width, seat.Height, s.Width, s.Height, h.BrowserZoom())
	return &mygo.Rectangle{X: seat.X + x, Y: seat.Y + y, Width: w, Height: height}
}

func (b *Controller) toolControls(c *ui.Context, t *Tab, state desktop.State) {
	if t == nil || t.Document != nil || t.HTML != nil || t.Media != nil {
		return
	}
	s := &t.tools.settings
	if s.Width == 0 {
		s.setSize(375, 667, "iPhone SE")
		s.Scheme = "system"
	}
	busy := t.tools.operation != nil || b.operation != nil || b.recording != nil
	disabled := t.Host == nil || state.Loading || busy || b.current() != t
	supported := !disabled && t.Host.BrowserToolsAvailable()
	appearanceSupported := !disabled && t.Host.BrowserColorSchemeAvailable()
	cookieSupported := !disabled && t.Host.BrowserCookieToolsAvailable()
	siteDataSupported := !disabled && t.Host.BrowserSiteDataToolsAvailable()
	devToolsSupported := !disabled && t.Host.BrowserDevToolsAvailable()
	devToolsLabel, devToolsHint, _ := browserDevToolsCopy(runtime.GOOS)
	zoom := 1.0
	if t.Host != nil {
		zoom = t.Host.BrowserZoom()
	}
	ui.ScrollHorizontal(c).Height(60).FillWidth().Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Row(c).Height(28).Gap(6).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, browserChromeLabel("设备尺寸", "Device size")).Disabled(disabled).Clicked() {
					b.CancelAction()
					s.Device = !s.Device
					b.showHost(t.Host)
					b.update()
				}
				if ui.Button(c, "−").Label(browserChromeLabel("缩小网页", "Zoom out")).Disabled(disabled || zoom <= 0.25).Clicked() {
					b.changeBrowserZoom(t, nextBrowserZoom(zoom, -1))
				}
				if ui.Button(c, fmt.Sprintf("%.0f%%", zoom*100)).Label(browserChromeLabel("重置网页缩放", "Reset zoom")).Disabled(disabled).Clicked() {
					b.changeBrowserZoom(t, 1)
				}
				if ui.Button(c, "+").Label(browserChromeLabel("放大网页", "Zoom in")).Disabled(disabled || zoom >= 5).Clicked() {
					b.changeBrowserZoom(t, nextBrowserZoom(zoom, 1))
				}
				ui.MenuButton(c, browserChromeLabel("网页外观", "Page appearance"), func(m *ui.Menu) {
					for _, choice := range []struct{ label, value string }{{browserChromeLabel("跟随系统", "System"), "system"}, {browserChromeLabel("浅色", "Light"), "light"}, {browserChromeLabel("深色", "Dark"), "dark"}} {
						if m.Item(choice.label).Checked(s.Scheme == choice.value).Disabled(!appearanceSupported).Chosen() {
							b.startBrowserTool(t, "scheme", choice.value)
						}
					}
				}).Disabled(!appearanceSupported).Tooltip(browserChromeLabel("覆盖当前页签的 prefers-color-scheme；不修改网页 CSS", "Override this tab’s prefers-color-scheme without modifying webpage CSS"))
				if ui.Button(c, browserChromeLabel("强制刷新", "Force reload")).Disabled(disabled).Clicked() {
					b.CancelAction()
					if err := t.Host.HardReloadBrowser(); err != nil {
						t.Error = err.Error()
					}
				}
				if ui.Button(c, browserChromeLabel("清 HTTP 缓存…", "Clear HTTP cache…")).Disabled(!supported).Clicked() {
					b.CancelAction()
					t.tools.confirmation = &browserDataConfirmation[*desktop.Host]{kind: "cache", host: t.Host, epoch: t.Host.BrowserPageVersion()}
					b.update()
				}
				if ui.Button(c, browserChromeLabel("清 Browser 登录 Cookie…", "Clear Browser sign-in cookies…")).Disabled(!cookieSupported).Tooltip(browserChromeLabel("仅当前标签所属分区的全部 Cookie；需确认，可能退出同分区页签登录。其他工作区/插件不受影响，保留站点存储、Service Worker 和 HTTP 缓存。", "All cookies in this tab’s profile only; confirmation required. Tabs in that profile may be signed out. Other workspaces/plugins are unaffected; site storage, service workers and HTTP cache are retained.")).Clicked() {
					b.CancelAction()
					t.tools.confirmation = &browserDataConfirmation[*desktop.Host]{kind: "cookies", host: t.Host, epoch: t.Host.BrowserPageVersion()}
					b.update()
				}
				if ui.Button(c, browserChromeLabel("清五类站点数据…", "Clear five site-data categories…")).Disabled(!siteDataSupported).Tooltip(browserChromeLabel("需当前平台支持精确站点数据清理。仅清当前分区所有站点的 Cookie、localStorage、IndexedDB、WebSQL 和 Service Worker；需确认，保留 HTTP 缓存和 CacheStorage，其他工作区/插件不受影响。", "Requires precise site-data clearing on this platform. Clears cookies, localStorage, IndexedDB, WebSQL and service workers for all sites in this profile, with confirmation. HTTP cache and CacheStorage are retained; other workspaces/plugins are unaffected.")).Clicked() {
					b.CancelAction()
					t.tools.confirmation = &browserDataConfirmation[*desktop.Host]{kind: "site-data", host: t.Host, epoch: t.Host.BrowserPageVersion()}
					b.update()
				}
				if ui.Button(c, devToolsLabel).Disabled(!devToolsSupported).Tooltip(devToolsHint).Clicked() {
					b.openBrowserDevTools(t)
				}
				if b.onMiniRestore != nil {
					label := browserChromeLabel("交互小浮窗", "Interactive mini-window")
					if b.mini != nil {
						label = browserChromeLabel("恢复原面板", "Restore original panel")
					}
					if ui.Button(c, label).Disabled(t.Host == nil).Clicked() {
						if b.mini != nil {
							b.restoreMiniPlayer()
						} else {
							b.openMiniPlayer(t)
						}
					}
				}
				if b.pip != nil {
					if ui.Button(c, browserChromeLabel("关闭画中画", "Close picture-in-picture")).Clicked() {
						b.stopPictureInPicture()
					}
				} else if ui.Button(c, browserChromeLabel("画中画", "Picture-in-picture")).Disabled(disabled).Clicked() {
					b.openPictureInPicture(t)
				}
				if t.tools.operation != nil && ui.Button(c, browserChromeLabel("取消等待", "Cancel waiting")).Clicked() {
					b.cancelBrowserTools()
					t.ActionMessage = browserChromeLabel("已取消等待；已提交的浏览器操作可能已生效", "Stopped waiting; the submitted browser operation may already have taken effect")
				}
			})
			ui.Row(c).Height(28).Gap(6).AlignItems(ui.Center).Children(func() {
				if confirm := t.tools.confirmation; confirm != nil {
					notice := browserChromeLabel("清除当前标签所属分区的全部 HTTP 缓存（含同分区页签）；保留 Cookies/站点数据，其他工作区/插件不受影响。继续？", "Clear all HTTP cache in this tab’s profile, including other tabs in the same profile? Cookies and site data are retained; other workspaces/plugins are unaffected.")
					unavailable := !supported
					if confirm.kind == "cookies" {
						notice = browserCookieNotice()
						unavailable = !cookieSupported
					} else if confirm.kind == "site-data" {
						notice = browserSiteDataNotice()
						unavailable = !siteDataSupported
					}
					notice = browserChromeLabel("分区「", "Profile “") + t.browserWorkspaceLabel() + "」：" + notice
					ui.Text(c, notice).FontSize(fonttheme.InterfaceSize(11)).Tooltip(notice)
					if ui.Button(c, browserChromeLabel("确认清除", "Confirm clear")).Key(confirm).Disabled(disabled || unavailable).Clicked() {
						b.startBrowserTool(t, confirm.kind, "")
					}
					if ui.Button(c, browserChromeLabel("取消", "Cancel")).Clicked() {
						t.tools.confirmation = nil
					}
					return
				}
				if !s.Device {
					ui.Text(c, browserChromeLabel("设备尺寸调整真实网页视口；网页外观需 Windows WebView2 或 macOS WKWebView；Linux 跟随系统。", "Device size changes the real webpage viewport. Appearance overrides require Windows WebView2 or macOS WKWebView; Linux follows the system.")).FontSize(fonttheme.InterfaceSize(11))
					return
				}
				label := s.Preset
				if label == "" {
					label = browserChromeLabel("自定义", "Custom")
				}
				ui.MenuButton(c, label, func(m *ui.Menu) {
					for _, preset := range browserDevicePresets {
						if m.Item(fmt.Sprintf("%s · %d × %d", preset.Name, preset.Width, preset.Height)).Checked(s.Preset == preset.Name).Disabled(disabled).Chosen() {
							b.updateBrowserDevice(t, preset.Width, preset.Height, preset.Name)
						}
					}
				}).Disabled(disabled)
				submitted := ui.TextInput(c, &s.WidthDraft).Width(65).Label(browserChromeLabel("视口宽度", "Viewport width")).Disabled(disabled).Submitted()
				ui.Text(c, "×")
				submitted = ui.TextInput(c, &s.HeightDraft).Width(65).Label(browserChromeLabel("视口高度", "Viewport height")).Disabled(disabled).Submitted() || submitted
				if ui.Button(c, browserChromeLabel("应用尺寸", "Apply size")).Disabled(disabled).Clicked() || submitted {
					w, h, err := s.dimensions(s.WidthDraft, s.HeightDraft)
					if err != nil {
						t.Error = err.Error()
					} else {
						b.updateBrowserDevice(t, w, h, "")
					}
				}
				lock := browserChromeLabel("锁定比例", "Lock aspect ratio")
				if s.AspectLocked {
					lock = browserChromeLabel("解除比例锁", "Unlock aspect ratio")
				}
				if ui.Button(c, lock).Disabled(disabled).Clicked() {
					s.AspectLocked = !s.AspectLocked
				}
				if ui.Button(c, browserChromeLabel("旋转", "Rotate")).Disabled(disabled).Clicked() {
					b.updateBrowserDevice(t, s.Height, s.Width, s.Preset)
				}
				ui.Text(c, browserChromeLabel("大尺寸会缩小至面板可见范围；不模拟触摸/UA", "Large sizes scale to fit the panel; touch and user agent are not emulated")).FontSize(fonttheme.InterfaceSize(11))
			})
		})
	})
}
