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
	profileSeen      bool
	floatingDocument *floatingDocumentWindow
	pip              *browserPictureInPicture
	rendered         bool
	actionTarget     func() ActionTarget
	operation        *previewOperation
	viewport         *mygo.Rectangle
	Tabs             []*Tab
	Active           int
	next             int
	parent           *mygo.Window
}
type Tab struct {
	tools       browserTools
	Document    *pdfpreview.Document
	HTML        *htmldocument.Document
	Media       *mediapreview.Document
	interactive *interactivePreview

	ActionMessage string
	ArtifactPath  string
	ID            int
	Draft, Error  string
	lastURL       string
	Host          *desktop.Host
}

const BrowserChromeHeight = 208

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
	b.Tabs = append(b.Tabs, &Tab{ID: b.next})
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
	if t.Document != nil || t.HTML != nil || t.Media != nil {
		b.NewTab(raw)
		return
	}
	t.Draft = raw
	normalized := NormalizeURL(raw)
	if normalized == "" {
		t.Error = "请输入有效的 HTTP(S) 地址"
		b.update()
		return
	}
	t.Draft = normalized
	t.Error = ""
	if t.Host != nil && t.Host.Alive() {
		if err := t.Host.Navigate(normalized); err != nil {
			t.Error = err.Error()
		}
		b.showHost(t.Host)
		return
	}
	h, err := desktop.NewHost(desktop.HostOptions{Parent: b.parent, URL: normalized, Title: "Browser", BrowserProfile: true, TopInset: chrome.TitleBarHeight + BrowserChromeHeight, Untrusted: true, OnChange: b.update})
	if err != nil {
		t.Error = err.Error()
		b.update()
		return
	}
	t.Host = h
	t.tools.settings.Scheme = "system"
	b.showHost(h)
	b.update()
}
func (b *Controller) Hide() {
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
	if t := b.current(); t.Host != nil {
		b.showHost(t.Host)
		if t.Document == nil && t.HTML == nil && t.Media == nil {
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
	if b.current() == nil {
		b.NewTab("")
	}
	t := b.current()
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
				if ui.Button(c, "返回工作区").Clicked() {
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
							} else if tab.Host != nil {
								st := tab.Host.State()
								if st.Title != "" {
									label += " " + truncate(st.Title, 12)
								}
							}
							if i == b.Active {
								label = "● " + label
							}
							if ui.Button(c, label).Clicked() {
								b.Select(i)
							}
						}
					})
				})
				if ui.Button(c, "+ 新标签").Clicked() {
					b.NewTab("")
				}
				if ui.Button(c, "关闭标签").Clicked() {
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
				if ui.Button(c, "←").Disabled(!s.CanBack).Clicked() && t.Host != nil {
					b.CancelAction()
					t.Host.Back()
				}
				if ui.Button(c, "→").Disabled(!s.CanForward).Clicked() && t.Host != nil {
					b.CancelAction()
					t.Host.Forward()
				}
				if s.Loading {
					if ui.Button(c, "停止").Clicked() {
						b.CancelAction()
						t.Host.Stop()
					}
				} else if ui.Button(c, "刷新").Disabled(t.Host == nil).Clicked() {
					b.CancelAction()
					t.Host.Reload()
				}
				if ui.TextInput(c, &t.Draft).Placeholder("https://example.com").Grow(1).Submitted() {
					b.Navigate(t.Draft)
				}
				if ui.Button(c, "前往").Clicked() {
					b.Navigate(t.Draft)
				}
				if ui.Button(c, "复制链接").Clicked() {
					raw := s.URL
					if raw == "" {
						raw = t.Draft
					}
					mygo.Clipboard.WriteText(raw)
				}
			})
			b.actionControls(c, t, s)
			b.toolControls(c, t, s)
			err := t.Error
			if err == "" {
				err = s.Error
			}
			status := s.URL
			if t.Media != nil {
				status = t.Media.Name + " · 媒体快照；只允许本文件，脚本/外网禁用；无法播放时请使用 Files 外部打开"
			}
			if t.Document != nil {
				status = t.Document.Name + " · PDF 只读快照；外部导航和脚本已禁用"
			}
			if t.HTML != nil {
				status = t.HTML.Name + " · " + t.HTML.Resources.Summary() + "；静态预览，脚本/外网禁用"
			}
			if status == "" {
				status = "每个标签保留独立页面和历史；网页无法调用应用权限。"
			}
			if s.Loading {
				status = "加载中 · " + status
			}
			if t.ActionMessage != "" {
				status = t.ActionMessage
			}
			if err != "" {
				status = err
			}
			if !b.profileSeen && t.Document == nil && t.HTML == nil && t.Media == nil && err == "" {
				const notice = "Browser 已启用独立登录区，首次需要重新登录；旧共享数据未迁移或删除。"
				ui.Row(c).Height(22).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, notice).FontSize(11).TextColor(chrome.Color(colors.Muted)).Grow(1).Tooltip(notice)
					if ui.Button(c, "知道了").Clicked() {
						b.profileSeen = true
					}
				})
			} else {
				ui.Text(c, truncate(status, 140)).FontSize(11).TextColor(chrome.Color(colors.Muted)).Tooltip(status)
			}
		})
		canvas := ui.Box(c).Key("browser-device-canvas").Fill().Grow(1).MinHeight(0).Background(chrome.Color(colors.Field))
		canvas.Children(func() { b.deviceResizeRails(c, t, canvas, chrome.Color(colors.Border)) })
	})
}
func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return string(r)
}
