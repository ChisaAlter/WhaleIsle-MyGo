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
	settings     browserToolSettings
	operation    *browserToolOperation
	confirmation *browserCacheConfirmation
}
type browserToolOperation struct {
	host   *desktop.Host
	epoch  uint64
	ctx    context.Context
	cancel context.CancelFunc
}
type browserCacheConfirmation struct {
	host  *desktop.Host
	epoch uint64
}

func (b *Controller) browserToolsCurrent(t *Tab, h *desktop.Host, epoch uint64) bool {
	return t != nil && b.current() == t && t.Document == nil && t.HTML == nil && t.Host == h && h != nil && h.Alive() && h.BrowserPageVersion() == epoch
}
func (b *Controller) cancelBrowserTools() {
	for _, t := range b.Tabs {
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
	if kind == "cache" {
		confirm := t.tools.confirmation
		if confirm == nil || confirm.host != h || confirm.epoch != epoch {
			return
		}
	}
	b.CancelAction()
	if !b.browserToolsCurrent(t, h, epoch) {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	op := &browserToolOperation{h, epoch, ctx, cancel}
	t.tools.operation = op
	t.Error = ""
	t.ActionMessage = "正在应用网页外观…"
	if kind == "cache" {
		t.ActionMessage = "正在清除本应用共享 WebView 配置的 HTTP 缓存…"
	}
	b.update()
	go func() {
		var err error
		if kind == "cache" {
			err = h.ClearBrowserHTTPCache(ctx)
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
				}
				return
			}
			if kind == "cache" {
				t.ActionMessage = "共享 WebView 配置的 HTTP 缓存已清除；Cookies 和站点数据保留"
			} else {
				t.tools.settings.Scheme = scheme
				t.ActionMessage = "网页 prefers-color-scheme 已更新"
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
	if t == nil || t.Host != h || t.Document != nil || t.HTML != nil || !t.tools.settings.Device {
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
	x, y, w, height, _ := fitBrowserDevice(seat.Width, seat.Height, s.Width, s.Height, h.BrowserZoom())
	return &mygo.Rectangle{X: seat.X + x, Y: seat.Y + y, Width: w, Height: height}
}

func (b *Controller) toolControls(c *ui.Context, t *Tab, state desktop.State) {
	if t == nil || t.Document != nil || t.HTML != nil {
		return
	}
	s := &t.tools.settings
	if s.Width == 0 {
		s.setSize(375, 667, "iPhone SE")
		s.Scheme = "system"
	}
	busy := t.tools.operation != nil || b.operation != nil
	disabled := t.Host == nil || state.Loading || busy || b.current() != t
	supported := !disabled && t.Host.BrowserToolsAvailable()
	zoom := 1.0
	if t.Host != nil {
		zoom = t.Host.BrowserZoom()
	}
	ui.ScrollHorizontal(c).Height(60).FillWidth().Children(func() {
		ui.Column(c).Gap(4).Children(func() {
			ui.Row(c).Height(28).Gap(6).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "设备尺寸").Disabled(disabled).Clicked() {
					b.CancelAction()
					s.Device = !s.Device
					b.showHost(t.Host)
					b.update()
				}
				if ui.Button(c, "−").Label("缩小网页").Disabled(disabled || zoom <= 0.25).Clicked() {
					b.changeBrowserZoom(t, nextBrowserZoom(zoom, -1))
				}
				if ui.Button(c, fmt.Sprintf("%.0f%%", zoom*100)).Label("重置网页缩放").Disabled(disabled).Clicked() {
					b.changeBrowserZoom(t, 1)
				}
				if ui.Button(c, "+").Label("放大网页").Disabled(disabled || zoom >= 5).Clicked() {
					b.changeBrowserZoom(t, nextBrowserZoom(zoom, 1))
				}
				ui.MenuButton(c, "网页外观", func(m *ui.Menu) {
					for _, choice := range []struct{ label, value string }{{"跟随系统", "system"}, {"浅色", "light"}, {"深色", "dark"}} {
						if m.Item(choice.label).Checked(s.Scheme == choice.value).Disabled(!supported).Chosen() {
							b.startBrowserTool(t, "scheme", choice.value)
						}
					}
				}).Disabled(!supported).Tooltip("覆盖当前页签的 prefers-color-scheme；不修改网页 CSS")
				if ui.Button(c, "强制刷新").Disabled(disabled).Clicked() {
					b.CancelAction()
					if err := t.Host.HardReloadBrowser(); err != nil {
						t.Error = err.Error()
					}
				}
				if ui.Button(c, "清 HTTP 缓存…").Disabled(!supported).Clicked() {
					b.CancelAction()
					t.tools.confirmation = &browserCacheConfirmation{t.Host, t.Host.BrowserPageVersion()}
					b.update()
				}
				if t.tools.operation != nil && ui.Button(c, "取消等待").Clicked() {
					b.cancelBrowserTools()
					t.ActionMessage = "已取消等待；已提交的浏览器操作可能已生效"
				}
			})
			ui.Row(c).Height(28).Gap(6).AlignItems(ui.Center).Children(func() {
				if t.tools.confirmation != nil {
					ui.Text(c, "清除本应用共享 WebView 配置的全部 HTTP 缓存（含其他页签/插件）；保留 Cookies/站点数据。继续？").FontSize(11)
					if ui.Button(c, "确认清除").Disabled(disabled).Clicked() {
						b.startBrowserTool(t, "cache", "")
					}
					if ui.Button(c, "取消").Clicked() {
						t.tools.confirmation = nil
					}
					return
				}
				if !s.Device {
					ui.Text(c, "设备尺寸调整真实网页视口；网页外观需 Windows WebView2。").FontSize(11)
					return
				}
				label := s.Preset
				if label == "" {
					label = "自定义"
				}
				ui.MenuButton(c, label, func(m *ui.Menu) {
					for _, preset := range browserDevicePresets {
						if m.Item(fmt.Sprintf("%s · %d × %d", preset.Name, preset.Width, preset.Height)).Checked(s.Preset == preset.Name).Disabled(disabled).Chosen() {
							b.updateBrowserDevice(t, preset.Width, preset.Height, preset.Name)
						}
					}
				}).Disabled(disabled)
				submitted := ui.TextInput(c, &s.WidthDraft).Width(65).Label("视口宽度").Disabled(disabled).Submitted()
				ui.Text(c, "×")
				submitted = ui.TextInput(c, &s.HeightDraft).Width(65).Label("视口高度").Disabled(disabled).Submitted() || submitted
				if ui.Button(c, "应用尺寸").Disabled(disabled).Clicked() || submitted {
					w, h, err := s.dimensions(s.WidthDraft, s.HeightDraft)
					if err != nil {
						t.Error = err.Error()
					} else {
						b.updateBrowserDevice(t, w, h, "")
					}
				}
				lock := "锁定比例"
				if s.AspectLocked {
					lock = "解除比例锁"
				}
				if ui.Button(c, lock).Disabled(disabled).Clicked() {
					s.AspectLocked = !s.AspectLocked
				}
				if ui.Button(c, "旋转").Disabled(disabled).Clicked() {
					b.updateBrowserDevice(t, s.Height, s.Width, s.Preset)
				}
				ui.Text(c, "大尺寸会缩小至面板可见范围；不模拟触摸/UA").FontSize(11)
			})
		})
	})
}
