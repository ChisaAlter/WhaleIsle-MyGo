package browserpreview

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pdfpreview"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Controller owns isolated native pages. It is used only from the UI thread;
// page state is owned by Host and its native navigation callbacks.
type Controller struct {
	mini                      *browserMiniPlayer
	miniGeometry              mygo.Rectangle
	onMiniOpen, onMiniRestore func()
	discovery                 *localDiscovery
	developerTools            func() bool
	workspaceOwner            func() WorkspaceOwner
	documentLoader            documentLoader
	onFocus                   func()
	profileSeen               bool
	floatingDocument          *floatingDocumentWindow
	pip                       *browserPictureInPicture
	recording                 *browserRecording
	rendered                  bool
	actionTarget              func() ActionTarget
	operation                 *previewOperation
	viewport                  *mygo.Rectangle
	Tabs                      []*Tab
	Active                    int
	next                      int
	parent                    *mygo.Window
}
type Tab struct {
	pdfPosition    *PDFReadingPosition
	pdfPageDraft   string
	workspace      browserWorkspace
	source         *DocumentSource
	sourcePending  bool
	floatAfterRead bool
	sourceRead     *documentRead
	tools          browserTools
	Document       *pdfpreview.Document
	HTML           *htmldocument.Document
	Media          *mediapreview.Document
	interactive    *interactivePreview

	ActionMessage string
	ArtifactPath  string
	RecordingPath string
	ID            int
	Draft, Error  string
	lastURL       string
	resumeURL     string
	Host          *desktop.Host
}

const BrowserChromeHeight = 242

func NewController(parent *mygo.Window) *Controller { return &Controller{parent: parent} }
func (b *Controller) current() *Tab {
	if b.Active < 0 || b.Active >= len(b.Tabs) {
		return nil
	}
	return b.Tabs[b.Active]
}
func (b *Controller) update() {
	if b.parent != nil {
		b.parent.Update(func() {})
	}
}
func (b *Controller) NewTab(raw string) {
	b.Hide()
	b.next++
	workspace, _ := b.browserWorkspace()
	b.Tabs = append(b.Tabs, &Tab{ID: b.next, workspace: workspace})
	b.Active = len(b.Tabs) - 1
	if raw != "" {
		b.Navigate(raw)
	}
	b.update()
}
func (b *Controller) Navigate(raw string) {
	b.CancelAction()
	if b.current() == nil {
		b.NewTab("")
	}
	t := b.current()
	workspace, scopeErr := b.browserWorkspace()
	if scopeErr != nil {
		t.Error = scopeErr.Error()
		b.update()
		return
	}
	if t.source != nil || t.Document != nil || t.HTML != nil || t.Media != nil || t.workspace.needsNewTab(workspace) {
		// Existing tabs keep their cookies/history in their original partition.
		b.NewTab("")
		t = b.current()
	}
	t.workspace = workspace
	t.Draft = raw
	normalized := NormalizeURL(raw)
	if normalized == "" {
		t.Error = browserChromeLabel("请输入有效的 HTTP(S) 地址", "Enter a valid HTTP(S) address")
		b.update()
		return
	}
	t.Draft = normalized
	t.Error = ""
	if t.Host != nil && t.Host.Alive() {
		if state := t.Host.State(); state.URL == normalized && !state.Loading && state.Error == "" {
			// ElectronWebViewImpl.loadUrl reloads an already observed address;
			// submitting it again must not append a duplicate history entry.
			t.Host.Reload()
		} else if err := t.Host.Navigate(normalized); err != nil {
			t.Error = err.Error()
		}
		b.showHost(t.Host)
		return
	}
	h, err := desktop.NewHost(desktop.HostOptions{Parent: b.parent, URL: normalized, Title: "Browser", BrowserProfile: true, BrowserPartition: t.workspace.key, TopInset: chrome.TitleBarHeight + BrowserChromeHeight, Untrusted: true, OnChange: b.update, OnFocus: b.focus})
	if err != nil {
		t.Error = err.Error()
		b.update()
		return
	}
	t.Host = h
	t.resumeURL = ""
	t.tools.settings.Scheme = "system"
	b.showHost(h)
	b.update()
}
func (b *Controller) Hide() {
	b.stopLocalDiscovery()
	b.pauseDocumentReads()
	b.CancelAction()
	b.cancelInteractivePreviews()
	for _, t := range b.Tabs {
		if t.Host != nil {
			t.Host.SetVisible(false)
		}
	}
}
func (b *Controller) Select(index int) {
	if index < 0 || index >= len(b.Tabs) {
		return
	}
	b.Hide()
	b.Active = index
	b.resumeCurrent()
	if t := b.current(); t.Host != nil {
		b.showHost(t.Host)
		if t.source == nil && t.Document == nil && t.HTML == nil && t.Media == nil {
			t.Draft = t.Host.State().URL
		}
	}
	b.update()
}
func (b *Controller) CloseTab(index int) {
	if index < 0 || index >= len(b.Tabs) {
		return
	}
	b.Hide()
	if b.mini != nil && b.mini.tab == b.Tabs[index] {
		b.CloseMiniPlayer()
	}
	if b.floatingDocument != nil && b.floatingDocument.tab == b.Tabs[index] {
		b.stopFloatingDocument()
	}
	if d := b.Tabs[index].Document; d != nil {
		d.Close()
	}
	if d := b.Tabs[index].Media; d != nil {
		d.Close()
	}
	if d := b.Tabs[index].HTML; d != nil {
		d.Close()
	}
	if h := b.Tabs[index].Host; h != nil {
		_ = h.Close()
	}
	b.Tabs = append(b.Tabs[:index], b.Tabs[index+1:]...)
	if index < b.Active {
		b.Active--
	}
	if b.Active >= len(b.Tabs) {
		b.Active = len(b.Tabs) - 1
	}
	if len(b.Tabs) == 0 {
		b.NewTab("")
	} else {
		b.Select(b.Active)
	}
}
func (b *Controller) Close() {
	b.CloseMiniPlayer()
	b.stopLocalDiscovery()
	b.pauseDocumentReads()
	b.stopFloatingDocument()
	b.stopPictureInPicture()
	b.CancelAction()
	b.cancelInteractivePreviews()
	for _, t := range b.Tabs {
		if t.Media != nil {
			t.Media.Close()
		}
		if t.HTML != nil {
			t.HTML.Close()
		}
		if t.Document != nil {
			t.Document.Close()
		}
		if t.Host != nil {
			_ = t.Host.Close()
		}
	}
	b.Tabs = nil
	b.Active = 0
}
func (b *Controller) View(c *ui.Context, themeID string, dark bool, onClose func()) {
	b.viewport = nil
	b.view(c, themeID, dark, onClose)
}
func (b *Controller) view(c *ui.Context, themeID string, dark bool, onClose func()) {
	b.rendered = true
	b.PruneActions()
	b.resumeCurrent()
	if b.current() == nil {
		b.NewTab("")
	}
	t := b.current()
	b.resumeLocalDiscovery(t)
	colors := theme.Resolve(themeID).Active(dark)
	s := desktop.State{}
	if t.Host != nil {
		s = t.Host.State()
		if t.Document == nil && t.HTML == nil && t.Media == nil && s.URL != "" && s.URL != t.lastURL {
			t.Draft = s.URL
			t.lastURL = s.URL
		}
		t.Host.SyncBounds()
		b.showHost(t.Host)
	}
	ui.Column(c).Fill().MinHeight(0).Background(chrome.Color(colors.BG)).Children(func() {
		ui.Column(c).Height(BrowserChromeHeight).Padding(6, 12, 6, 12).Gap(4).Children(func() {
			ui.Row(c).Height(30).Gap(6).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, browserChromeLabel("返回工作区", "Back to workspace")).Clicked() {
					b.Hide()
					if onClose != nil {
						onClose()
					}
				}
				// A horizontal scroller keeps every tab reachable instead of clipping.
				ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).Gap(4).Children(func() {
						for i, tab := range b.Tabs {
							label := fmt.Sprintf("%d", tab.ID)
							if tab.Document != nil {
								label += " " + truncate(tab.Document.Name, 12)
							} else if tab.Media != nil {
								label += " " + truncate(tab.Media.Name, 12)
							} else if tab.HTML != nil {
								label += " " + truncate(tab.HTML.Name, 12)
							} else if tab.source != nil {
								label += " " + truncate(tab.source.Path, 12)
							} else if tab.Host != nil {
								st := tab.Host.State()
								if st.Title != "" {
									label += " " + truncate(st.Title, 12)
								}
							}
							if i == b.Active {
								label = "● " + label
							}
							if ui.Button(c, label).Tooltip(browserChromeLabel("Browser 分区：", "Browser profile: ") + tab.browserWorkspaceLabel()).Clicked() {
								b.Select(i)
							}
						}
					})
				})
				if ui.Button(c, browserChromeLabel("+ 新标签", "+ New tab")).Clicked() {
					b.NewTab("")
				}
				if ui.Button(c, browserChromeLabel("关闭标签", "Close tab")).Clicked() {
					b.CloseTab(b.Active)
				}
			})
			ui.Row(c).Height(32).Gap(6).AlignItems(ui.Center).Children(func() {
				if t.Media != nil {
					b.mediaControls(c, t)
					return
				}
				if t.Document != nil {
					b.documentControls(c, t)
					return
				}
				if t.HTML != nil {
					b.htmlControls(c, t, dark)
					return
				}
				if t.source != nil {
					b.documentSourceControls(c, t)
					return
				}
				if ui.Button(c, "←").Disabled(!s.CanBack).Clicked() && t.Host != nil {
					b.CancelAction()
					t.Host.Back()
				}
				if ui.Button(c, "→").Disabled(!s.CanForward).Clicked() && t.Host != nil {
					b.CancelAction()
					t.Host.Forward()
				}
				if s.Loading {
					if ui.Button(c, browserChromeLabel("停止", "Stop")).Clicked() {
						b.CancelAction()
						t.Host.Stop()
					}
				} else if ui.Button(c, browserChromeLabel("刷新", "Reload")).Disabled(t.Host == nil && t.resumeURL == "").Clicked() {
					b.CancelAction()
					if t.resumeURL != "" {
						b.restoreBrowserAddress(t)
					} else {
						t.Host.Reload()
					}
				}
				if ui.TextInput(c, &t.Draft).Placeholder("https://example.com").Grow(1).Submitted() {
					b.Navigate(t.Draft)
				}
				if ui.Button(c, browserChromeLabel("前往", "Go")).Clicked() {
					b.Navigate(t.Draft)
				}
				if ui.Button(c, browserChromeLabel("系统浏览器打开", "Open in system browser")).Disabled(b.current() != t || NormalizeURL(t.Draft) == "").Tooltip(browserChromeLabel("打开地址栏中的 HTTP(S) 地址；使用系统浏览器自己的登录状态，不传递 Browser 分区 Cookie。", "Open the address bar HTTP(S) URL using the system browser’s own sign-in state. Browser profile cookies are not transferred.")).Clicked() && b.current() == t {
					if err := openBrowserAddress(t.Draft, desktop.OpenExternal); err != nil {
						t.Error = err.Error()
					} else {
						t.Error = ""
						t.ActionMessage = browserChromeLabel("已提交到系统浏览器；当前 Browser 页签和分区保持不变", "Sent to the system browser; this tab and its profile are unchanged")
					}
				}
				if ui.Button(c, browserChromeLabel("复制链接", "Copy URL")).Clicked() {
					raw := s.URL
					if raw == "" {
						raw = t.Draft
					}
					mygo.Clipboard.WriteText(raw)
				}
			})
			b.localServerControls(c, t)
			b.pdfReadingControls(c, t)
			b.actionControls(c, t, s)
			b.toolControls(c, t, s)
			err := t.Error
			if err == "" {
				err = s.Error
			}
			status := s.URL
			if t.Media != nil {
				status = t.Media.Name + browserChromeLabel(" · 媒体快照；只允许本文件，脚本/外网禁用；无法播放时请使用 Files 外部打开", " · Media snapshot; only this file is allowed, scripts/network disabled. Use Files → Open external if playback fails.")
			}
			if t.Document != nil {
				status = t.Document.Name + browserChromeLabel(" · PDF 只读快照；文档脚本与外部导航已禁用", " · Read-only PDF snapshot; document scripts and external navigation disabled")
			}
			if t.HTML != nil {
				status = t.HTML.Name + " · " + t.HTML.Resources.Summary() + browserChromeLabel("；静态预览，脚本/外网禁用", "; static preview, scripts/network disabled")
			}
			if status == "" {
				status = browserChromeLabel("每个标签保留独立页面和历史；网页无法调用应用权限。", "Each tab retains its own page and history; webpages cannot invoke app permissions.")
			}
			if s.Loading {
				status = browserChromeLabel("加载中 · ", "Loading · ") + status
			}
			if t.ActionMessage != "" {
				status = t.ActionMessage
			}
			if err != "" {
				status = err
			}
			if t.Document == nil && t.HTML == nil && t.Media == nil {
				status = browserChromeLabel("分区「", "Profile “") + t.browserWorkspaceLabel() + browserChromeLabel("」 · ", "” · ") + status
			}
			if !b.profileSeen && t.source == nil && t.Document == nil && t.HTML == nil && t.Media == nil && err == "" {
				notice := browserChromeLabel("Browser 按工作区分区；未归属工作区的会话使用各自独立分区。已有登录区保留；新分区首次需登录。", "Browser profiles are separated by workspace. Sessions outside a workspace have independent profiles. Existing profiles are retained; sign in once in each new profile.")
				ui.Row(c).Height(22).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, notice).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted)).Grow(1).Tooltip(notice)
					if ui.Button(c, browserChromeLabel("知道了", "Got it")).Clicked() {
						b.profileSeen = true
					}
				})
			} else {
				ui.Text(c, truncate(status, 140)).FontSize(theme.InterfaceSize(11)).TextColor(chrome.Color(colors.Muted)).Tooltip(status)
			}
		})
		canvas := ui.Box(c).Key("browser-device-canvas").Fill().Grow(1).MinHeight(0).Background(chrome.Color(colors.Field))
		canvas.Children(func() {
			if t.resumeURL != "" && t.Host == nil {
				ui.Column(c).Fill().AlignItems(ui.Center).Justify(ui.Center).Gap(10).Padding(20).Children(func() {
					ui.Text(c, browserChromeLabel("恢复上次浏览的页面", "Restore the previous page")).FontSize(theme.InterfaceSize(14))
					ui.Text(c, t.resumeURL).FontSize(theme.InterfaceSize(12)).Tooltip(t.resumeURL)
					if ui.Button(c, browserChromeLabel("恢复页面", "Restore page")).Clicked() {
						b.restoreBrowserAddress(t)
					}
				})
			}
			b.deviceResizeRails(c, t, canvas, chrome.Color(colors.Border))
		})
	})
}
func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return string(r)
}
