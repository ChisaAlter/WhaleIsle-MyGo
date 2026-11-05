package browserpreview

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

var browserBackIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="m9.5 4-4 4 4 4" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
var browserForwardIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="m6.5 4 4 4-4 4" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
var browserReloadIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M13 7a5 5 0 1 0-.6 3.5M13 3v4H9" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
var browserStopIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="m4.5 4.5 7 7m0-7-7 7" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/></svg>`))
var browserGoIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none"><path d="M3.5 8h9m-4-4 4 4-4 4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
var browserMoreIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><circle cx="4" cy="8" r="1" fill="currentColor"/><circle cx="8" cy="8" r="1" fill="currentColor"/><circle cx="12" cy="8" r="1" fill="currentColor"/></svg>`))

func browserChromeTool(c *ui.Context, key, label string, icon *ui.SVG) ui.Element {
	button := ui.Button(c.Key(key), "").Label(label).Tooltip(label).Size(28, 28).Shrink(0).Padding(0).Radius(6).
		BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center)
	if button.Hovered() {
		button.Background(c.Theme().SurfaceHover)
	}
	button.Children(func() { ui.Icon(c, icon).Size(16, 16).PassThrough() })
	return button
}

func browserWebTab(t *Tab) bool {
	return t != nil && t.source == nil && t.Document == nil && t.HTML == nil && t.Media == nil
}

func (b *Controller) chromeHeight() int {
	height := BrowserChromeHeight
	if b.docked {
		height = BrowserResourceChromeHeight
	}
	if t := b.current(); browserWebTab(t) && t.toolsExpanded {
		height += 30
	}
	if t := b.current(); browserWebTab(t) && t.tools.settings.Device {
		height += 32
	}
	return height
}

func (b *Controller) View(c *ui.Context, themeID string, dark bool, onClose func()) {
	b.docked = false
	b.viewSeat(c, themeID, dark, onClose)
}

func (b *Controller) viewSeat(c *ui.Context, themeID string, dark bool, onClose func()) {
	b.frameContext = c
	seat := ui.Column(c).Key("browser-resource-seat").Fill().MinWidth(0).MinHeight(0)
	r, height := seat.Bounds(), b.chromeHeight()
	b.viewport = &mygo.Rectangle{X: int(r.X), Y: int(r.Y) + height, Width: int(r.W), Height: max(0, int(r.H)-height)}
	seat.Children(func() { b.view(c, themeID, dark, onClose) })
}

func (b *Controller) view(c *ui.Context, themeID string, dark bool, onClose func()) {
	b.rendered = true
	b.PruneActions()
	b.resumeCurrent()
	if b.current() == nil {
		b.NewTab("")
	}
	t := b.current()
	if browserWebTab(t) && t.toolsExpanded {
		b.resumeLocalDiscovery(t)
	} else {
		b.stopLocalDiscovery()
	}
	colors := theme.Resolve(themeID).Active(dark)
	state := desktop.State{}
	if t.Host != nil {
		state = t.Host.State()
		if browserWebTab(t) && state.URL != "" && state.URL != t.lastURL {
			t.Draft, t.lastURL = state.URL, state.URL
		}
		t.Host.SyncBounds()
		b.showHost(t.Host)
	}
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(chrome.Color(colors.BG)).Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Height(float32(b.chromeHeight())).Shrink(0).Padding(6, 8).Gap(4).Children(func() {
			// DockKit owns the tabs and close control in a workspace resource seat.
			// The standalone browser keeps its own tab strip.
			if !b.docked {
				b.browserTabs(c, onClose)
			}
			ui.Row(c).FillWidth().MinWidth(0).Height(32).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
				switch {
				case t.Media != nil:
					b.mediaControls(c, t)
				case t.Document != nil:
					b.documentControls(c, t)
				case t.HTML != nil:
					b.htmlControls(c, t, dark)
				case t.source != nil:
					b.documentSourceControls(c, t)
				default:
					b.browserNavigation(c, t, state, onClose)
				}
			})
			if browserWebTab(t) && t.toolsExpanded {
				b.localServerControls(c, t)
			}
			if browserWebTab(t) {
				b.toolControls(c, t, state)
			}
			status := t.ActionMessage
			if status == "" {
				status = state.URL
			}
			if status == "" && t.source != nil {
				status = t.source.Path
			}
			problem := t.Error
			if problem == "" {
				problem = state.Error
			}
			ink := chrome.Color(colors.Muted)
			if problem != "" {
				status, ink = problem, chrome.Color(colors.Danger)
			} else if state.Loading {
				status = browserChromeLabel("正在打开…", "Loading…")
			}
			ui.Text(c, status).FillWidth().Height(18).MinWidth(0).SingleLine().Ellipsis("…").FontSize(theme.InterfaceSize(12)).TextColor(ink).Tooltip(status)
		})
		canvas := ui.Box(c).Key("browser-device-canvas").FillWidth().Grow(1).MinWidth(0).MinHeight(0).Background(chrome.Color(colors.Field))
		canvas.Children(func() {
			if t.Host == nil {
				b.browserPlaceholder(c, t, chrome.Color(colors.Muted))
			}
			b.deviceResizeRails(c, t, canvas, chrome.Color(colors.Border))
		})
		if browserWebTab(t) {
			b.browserDataDialog(c, t, state)
		}
	})
}

func (b *Controller) browserNavigation(c *ui.Context, t *Tab, state desktop.State, onClose func()) {
	if browserChromeTool(c, "browser-back", browserChromeLabel("后退", "Back"), browserBackIcon).Disabled(!state.CanBack).Clicked() && t.Host != nil {
		b.CancelAction()
		t.Error, t.ActionMessage = "", ""
		t.Host.Back()
	}
	if browserChromeTool(c, "browser-forward", browserChromeLabel("前进", "Forward"), browserForwardIcon).Disabled(!state.CanForward).Clicked() && t.Host != nil {
		b.CancelAction()
		t.Error, t.ActionMessage = "", ""
		t.Host.Forward()
	}
	if state.Loading {
		if browserChromeTool(c, "browser-stop", browserChromeLabel("停止加载", "Stop loading"), browserStopIcon).Clicked() && t.Host != nil {
			b.CancelAction()
			t.Host.Stop()
		}
	} else if browserChromeTool(c, "browser-reload", browserChromeLabel("刷新", "Reload"), browserReloadIcon).Disabled(t.Host == nil && t.resumeURL == "").Clicked() {
		b.CancelAction()
		t.Error, t.ActionMessage = "", ""
		if t.resumeURL != "" {
			b.restoreBrowserAddress(t)
		} else if t.Host != nil {
			t.Host.Reload()
		}
	}
	address := ui.TextInput(c, &t.Draft).Label(browserChromeLabel("网页地址", "Web address")).Placeholder(browserChromeLabel("输入 HTTP(S) 地址", "Enter an HTTP(S) address")).Grow(1).MinWidth(0).Height(28).Radius(6).FontSize(theme.InterfaceSize(12))
	goButton := browserChromeTool(c, "browser-go", browserChromeLabel("前往", "Go"), browserGoIcon)
	if address.Submitted() && !address.Composing() || goButton.Clicked() {
		b.Navigate(t.Draft)
	}
	currentURL := state.URL
	if currentURL == "" {
		currentURL = t.Draft
	}
	if clicked, err := externalOpenButton(c, currentURL, desktop.OpenExternal); clicked {
		if err != nil {
			t.Error = err.Error()
		} else {
			t.Error, t.ActionMessage = "", browserChromeLabel("已在系统浏览器打开", "Opened in the system browser")
		}
	}
	if b.onMiniRestore != nil {
		label := browserChromeLabel("交互小浮窗", "Interactive mini-window")
		if b.mini != nil {
			label = browserChromeLabel("恢复原面板", "Restore original panel")
		}
		if browserChromeTool(c, "browser-mini", label, previewMiniIcon).Disabled(t.Host == nil).Clicked() {
			if b.mini != nil {
				b.restoreMiniPlayer()
			} else {
				b.openMiniPlayer(t)
			}
		}
	}
	browserChromeTool(c, "browser-options", browserChromeLabel("浏览器选项", "Browser options"), browserMoreIcon).Menu(func(menu *ui.Menu) {
		if b.current() != t {
			return
		}
		if menu.Item(browserChromeLabel("复制链接", "Copy URL")).Disabled(strings.TrimSpace(currentURL) == "").Chosen() {
			mygo.Clipboard.WriteText(currentURL)
		}
		b.browserToolsMenu(menu, t, state)
		b.savedBrowserTabsMenu(menu)
		menu.Separator()
		menu.Item(browserChromeLabel("浏览器分区：", "Browser profile: ") + t.browserWorkspaceLabel()).Disabled(true)
		if onClose != nil && menu.Item(browserChromeLabel("关闭页面", "Close page")).Chosen() {
			b.Hide()
			onClose()
		}
	}).Key(t)
}

func (b *Controller) documentHeader(c *ui.Context, t *Tab, name string, externalPath func() (string, error), dark bool) {
	ui.Text(c, name).FontSize(theme.InterfaceSize(12)).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").Tooltip(name)
	if browserChromeTool(c, "document-float", browserChromeLabel("独立窗口预览", "Preview in separate window"), externalLinkIcon).Disabled(t.Host == nil || !t.Host.Alive()).Clicked() {
		b.openFloatingDocument(t)
	}
	browserChromeTool(c, "document-options", browserChromeLabel("文档选项", "Document options"), browserMoreIcon).Key(t).Menu(func(menu *ui.Menu) {
		if b.current() != t {
			return
		}
		if externalPath != nil {
			if menu.Item(browserChromeLabel("系统打开", "Open externally")).Chosen() {
				path, err := externalPath()
				if err == nil {
					err = mygo.Shell.OpenPath(path)
				}
				if err != nil {
					t.Error = err.Error()
				}
			}
			if menu.Item(browserChromeLabel("复制文件路径", "Copy file path")).Chosen() {
				path, err := externalPath()
				if err != nil {
					t.Error = err.Error()
				} else {
					mygo.Clipboard.WriteText(path)
				}
			}
		}
		if t.source != nil && menu.Item(browserChromeLabel("重新读取", "Reread")).Disabled(t.sourceRead != nil).Chosen() {
			b.retryDocument(t)
		}
		if t.HTML != nil && t.HTML.CanRunScripts() {
			b.interactiveHTMLMenu(menu, t, dark)
		}
		b.savedBrowserTabsMenu(menu)
	})
}

// Recovery may contain multiple pages from the earlier standalone browser.
// Docked resources omit the nested strip but keep those pages reachable.
func (b *Controller) savedBrowserTabsMenu(menu *ui.Menu) {
	if len(b.Tabs) < 2 {
		return
	}
	menu.Submenu(browserChromeLabel("已打开的页面", "Open pages"), func(sub *ui.Menu) {
		for index, tab := range b.Tabs {
			title := tab.Draft
			if tab.Host != nil && tab.Host.State().Title != "" {
				title = tab.Host.State().Title
			}
			if title == "" {
				title = browserChromeLabel("新标签", "New tab")
			}
			if sub.Item(title).Checked(index == b.Active).Chosen() {
				b.Select(index)
			}
		}
	})
}

func (b *Controller) browserTabs(c *ui.Context, onClose func()) {
	ui.Row(c).FillWidth().MinWidth(0).Height(32).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
		if onClose != nil && browserChromeTool(c, "browser-return", browserChromeLabel("返回工作区", "Back to workspace"), browserBackIcon).Clicked() {
			b.Hide()
			onClose()
		}
		ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(4).Children(func() {
				for i, tab := range b.Tabs {
					title := browserChromeLabel("新标签", "New tab")
					if tab.source != nil {
						title = filepath.Base(tab.source.Path)
					} else if tab.Host != nil && tab.Host.State().Title != "" {
						title = tab.Host.State().Title
					}
					button := ui.Button(c, truncate(title, 24)).Key(tab.ID).Height(28).MaxWidth(180).Tooltip(title)
					if i == b.Active {
						button.Background(c.Theme().SurfaceHover)
					}
					if button.Clicked() {
						b.Select(i)
					}
				}
			})
		})
		if ui.Button(c, browserChromeLabel("新标签", "New tab")).Height(28).Clicked() {
			b.NewTab("")
		}
		if browserChromeTool(c, "browser-close-tab", browserChromeLabel("关闭标签", "Close tab"), browserStopIcon).Clicked() {
			b.CloseTab(b.Active)
		}
	})
}

func (b *Controller) browserPlaceholder(c *ui.Context, t *Tab, muted ui.Color) {
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).AlignItems(ui.Center).Justify(ui.Center).Padding(24).Gap(12).Children(func() {
		switch {
		case t.resumeURL != "":
			ui.Text(c, browserChromeLabel("上次打开", "Restore the previous page")).FontSize(theme.InterfaceSize(14))
			ui.Text(c, t.resumeURL).FillWidth().MinWidth(0).SingleLine().Ellipsis("…").FontSize(theme.InterfaceSize(12)).TextColor(muted).Tooltip(t.resumeURL)
			if ui.Button(c, browserChromeLabel("恢复页面", "Restore page")).Clicked() {
				b.restoreBrowserAddress(t)
			}
		case t.source != nil:
			message := browserChromeLabel("正在读取文档…", "Reading document…")
			if t.Error != "" {
				message = t.Error
			}
			ui.Text(c, message).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if t.Error != "" && ui.Button(c, browserChromeLabel("重试", "Retry")).Clicked() {
				b.retryDocument(t)
			}
		default:
			ui.Text(c, browserChromeLabel("输入 HTTP(S) 地址开始浏览", "Enter an address to start browsing")).FontSize(theme.InterfaceSize(14)).TextColor(muted)
		}
	})
}
