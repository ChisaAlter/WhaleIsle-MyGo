package browserpreview

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"os"
	"runtime"
	"whaleisle/internal/desktop/browserrecording"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
)

var previewMiniIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><rect x="2" y="3" width="12" height="10" rx="1.5" stroke="currentColor" stroke-width="1.3"/><rect x="8" y="7" width="4.5" height="4.5" rx=".7" fill="currentColor"/></svg>`))

// The active installed ui-preview provider keeps tools in one anchored More
// menu. Only its explicitly enabled device toolbar occupies another row.
func (b *Controller) browserToolsMenu(menu *ui.Menu, t *Tab, state desktop.State) {
	s := &t.tools.settings
	if s.Width == 0 {
		s.setSize(375, 667, "iPhone SE")
		s.Scheme = "system"
	}
	disabled := t.Host == nil || state.Loading || t.tools.operation != nil || b.operation != nil || b.recording != nil
	toolAvailable := !disabled && t.Host.BrowserToolsAvailable()
	if menu.Item(browserChromeLabel("强制刷新", "Force reload")).Disabled(disabled).Chosen() {
		b.CancelAction()
		if err := t.Host.HardReloadBrowser(); err != nil {
			t.Error = err.Error()
		}
	}
	label, _, _ := browserDevToolsCopy(runtime.GOOS)
	if menu.Item(label).Disabled(disabled || !t.Host.BrowserDevToolsAvailable()).Chosen() {
		b.openBrowserDevTools(t)
	}
	pipLabel := browserPreviewText("pipOpen")
	if b.pip != nil {
		pipLabel = browserPreviewText("pipClose")
	}
	if menu.Item(pipLabel).Disabled(disabled).Chosen() {
		if b.pip != nil {
			b.stopPictureInPicture()
		} else {
			b.openPictureInPicture(t)
		}
	}
	deviceLabel := browserPreviewText("deviceToolbarShow")
	if s.Device {
		deviceLabel = browserPreviewText("deviceToolbarHide")
	}
	if menu.Item(deviceLabel).Checked(s.Device).Disabled(disabled).Chosen() {
		b.CancelAction()
		s.Device = !s.Device
		b.showHost(t.Host)
		b.update()
	}
	if menu.Item(browserChromeLabel("选择元素", "Select element")).Disabled(disabled).Chosen() {
		b.startAction(t, "pick")
	}
	b.recordingMenu(menu, t, disabled)
	if menu.Item(browserChromeLabel("网页截图", "Screenshot")).Disabled(disabled).Chosen() {
		b.startAction(t, "screenshot")
	}
	if menu.Item(browserChromeLabel("可视批注", "Visual annotation")).Disabled(disabled).Chosen() {
		b.startAnnotation(t)
	}
	if op := b.operation; op != nil {
		if op.ready {
			if menu.Item(browserChromeLabel("查看批注截图", "View annotated screenshot")).Chosen() {
				mygo.Shell.ShowItemInFolder(op.attachmentPath)
			}
			if menu.Item(browserChromeLabel("复制批注", "Copy annotation")).Chosen() {
				mygo.Clipboard.WriteText(op.annotation)
			}
			if menu.Item(browserChromeLabel("批注和截图加入草稿", "Add annotation and screenshot to draft")).Chosen() {
				b.appendAnnotation(op)
			}
		}
		if menu.Item(browserChromeLabel("取消当前操作", "Cancel current operation")).Chosen() {
			b.CancelAction()
		}
	}
	if t.ArtifactPath != "" && menu.Item(browserChromeLabel("显示截图", "Show screenshot")).Chosen() {
		if info, err := os.Stat(t.ArtifactPath); err == nil && info.Mode().IsRegular() {
			mygo.Shell.ShowItemInFolder(t.ArtifactPath)
		} else {
			t.Error = browserChromeLabel("截图文件已移除，请重新截图", "Screenshot file was removed; take another screenshot")
		}
	}
	menu.Separator()
	menu.Submenu(browserChromeLabel("网页外观", "Page appearance"), func(sub *ui.Menu) {
		for _, choice := range []struct{ label, value string }{{browserChromeLabel("跟随系统", "System"), "system"}, {browserChromeLabel("浅色", "Light"), "light"}, {browserChromeLabel("深色", "Dark"), "dark"}} {
			if sub.Item(choice.label).Checked(s.Scheme == choice.value).Disabled(disabled || !t.Host.BrowserColorSchemeAvailable()).Chosen() {
				b.startBrowserTool(t, "scheme", choice.value)
			}
		}
	})
	zoom := 1.0
	if t.Host != nil {
		zoom = t.Host.BrowserZoom()
	}
	menu.Separator()
	if menu.Item(browserChromeLabel("缩小", "Zoom out")).Disabled(disabled || zoom <= .25).Chosen() {
		b.changeBrowserZoom(t, nextBrowserZoom(zoom, -1))
	}
	menu.Item(fmt.Sprintf("%.0f%%", zoom*100)).Disabled(true)
	if menu.Item(browserChromeLabel("放大", "Zoom in")).Disabled(disabled || zoom >= 5).Chosen() {
		b.changeBrowserZoom(t, nextBrowserZoom(zoom, 1))
	}
	if menu.Item(browserChromeLabel("重置为100%", "Reset to 100%")).Disabled(disabled).Chosen() {
		b.changeBrowserZoom(t, 1)
	}
	menu.Separator()
	lockLabel := browserPreviewText("viewportLock")
	if s.AspectLocked {
		lockLabel = browserPreviewText("viewportUnlock")
	}
	if s.Device && menu.Item(lockLabel).Checked(s.AspectLocked).Chosen() {
		s.AspectLocked = !s.AspectLocked
	}
	for _, choice := range []struct {
		label, kind string
		available   bool
	}{{browserPreviewText("clearCookies"), "cookies", !disabled && t.Host.BrowserCookieToolsAvailable()}, {browserPreviewText("clearCache"), "cache", toolAvailable}, {browserChromeLabel("清五类站点数据…", "Clear five site-data categories…"), "site-data", !disabled && t.Host.BrowserSiteDataToolsAvailable()}} {
		if menu.Item(choice.label).Disabled(!choice.available).Chosen() {
			b.CancelAction()
			t.tools.confirmation = &browserDataConfirmation[*desktop.Host]{kind: choice.kind, host: t.Host, epoch: t.Host.BrowserPageVersion()}
			b.update()
		}
	}
	if t.tools.operation != nil && menu.Item(browserChromeLabel("取消等待", "Cancel waiting")).Chosen() {
		b.cancelBrowserTools()
		t.ActionMessage = browserChromeLabel("已取消等待；已提交的操作可能已生效", "Stopped waiting; the submitted operation may already have taken effect")
	}
	menu.Separator()
	if menu.Item(browserPreviewText("discovered")).Checked(t.toolsExpanded).Chosen() {
		t.toolsExpanded = !t.toolsExpanded
		b.update()
	}
}

func (b *Controller) recordingMenu(menu *ui.Menu, t *Tab, disabled bool) {
	if r := b.recording; r != nil && r.tab == t {
		switch r.phase {
		case "recording":
			if menu.Item(browserPreviewText("recordStop")).Chosen() {
				b.stopBrowserRecording(r)
			}
		case "ready":
			if menu.Item(browserChromeLabel("导出视频…", "Export video…")).Chosen() {
				b.exportBrowserRecording(r)
			}
		default:
			menu.Item(t.ActionMessage).Disabled(true)
		}
		if menu.Item(browserChromeLabel("取消录制", "Cancel recording")).Chosen() {
			b.cancelBrowserRecording(browserChromeLabel("录制已取消，未导出", "Recording cancelled; nothing exported"))
			b.update()
		}
	} else if menu.Item(browserPreviewText("recordStart")).Disabled(disabled || !browserrecording.PlatformAvailable()).Chosen() {
		b.startBrowserRecording(t)
	}
	if t.RecordingPath != "" && menu.Item(browserChromeLabel("显示视频", "Show video")).Chosen() {
		if info, err := os.Stat(t.RecordingPath); err == nil && info.Mode().IsRegular() {
			mygo.Shell.ShowItemInFolder(t.RecordingPath)
		} else {
			t.Error = browserChromeLabel("视频文件已移除，请重新录制", "Video file was removed; record again")
		}
	}
}

func (b *Controller) browserDataDialog(c *ui.Context, t *Tab, state desktop.State) {
	confirm := t.tools.confirmation
	if confirm == nil {
		return
	}
	open := true
	notice := browserChromeLabel("清除当前分区的全部 HTTP 缓存；保留 Cookie 与站点数据。", "Clear all HTTP cache in this profile; retain cookies and site data.")
	supported := t.Host != nil && t.Host.BrowserToolsAvailable()
	if confirm.kind == "cookies" {
		notice = browserCookieNotice()
		supported = t.Host != nil && t.Host.BrowserCookieToolsAvailable()
	} else if confirm.kind == "site-data" {
		notice = browserSiteDataNotice()
		supported = t.Host != nil && t.Host.BrowserSiteDataToolsAvailable()
	}
	ui.DialogBase(c.Key(confirm), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), 480, 20).MaxHeightPercent(88).Gap(12).Children(func() {
			ui.Text(c, browserChromeLabel("确认清理浏览器数据", "Clear browser data?")).FontSize(16).Bold()
			ui.Text(c, browserChromeLabel("分区：", "Profile: ")+t.browserWorkspaceLabel()).FontSize(12)
			ui.Text(c, notice).FontSize(12)
			ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, browserChromeLabel("取消", "Cancel")).Clicked() {
					open = false
				}
				if ui.PrimaryButton(c, browserChromeLabel("确认清除", "Confirm clear")).Disabled(!supported || state.Loading || t.tools.operation != nil || b.operation != nil || b.recording != nil).Clicked() {
					b.startBrowserTool(t, confirm.kind, "")
					open = false
				}
			})
		})
	})
	if !open {
		t.tools.confirmation = nil
	}
}
