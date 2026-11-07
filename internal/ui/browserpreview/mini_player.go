package browserpreview

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	desktop "whaleisle/internal/desktop/webview"
	"whaleisle/internal/ui/theme"
)

// The mini-player is a presentation seat for an existing live guest. It never
// creates a Host, navigates, captures frames, or transfers the browser profile.
type browserMiniPlayer struct {
	tab         *Tab
	host        *desktop.Host
	geometry    mygo.Rectangle
	initialized bool
	rendered    bool
	bounds      mygo.Rectangle
	context     *ui.Context
}

func miniPlayerGeometry(r, container mygo.Rectangle) mygo.Rectangle {
	r.Width = min(max(240, r.Width), max(1, container.Width-24))
	r.Height = min(max(150, r.Height), max(1, container.Height-24))
	r.X = min(max(12, r.X), max(12, container.Width-r.Width-12))
	r.Y = min(max(12, r.Y), max(12, container.Height-r.Height-12))
	return r
}
func resizeMiniPlayer(r mygo.Rectangle, dx, dy int, side string, container mygo.Rectangle) mygo.Rectangle {
	right, bottom := r.X+r.Width, r.Y+r.Height
	if strings.Contains(side, "w") {
		r.X = min(max(r.X+dx, 12), right-240)
	}
	if strings.Contains(side, "n") {
		r.Y = min(max(r.Y+dy, 12), bottom-150)
	}
	if strings.Contains(side, "e") {
		right = min(max(right+dx, r.X+240), max(12, container.Width-12))
	}
	if strings.Contains(side, "s") {
		bottom = min(max(bottom+dy, r.Y+150), max(12, container.Height-12))
	}
	r.Width, r.Height = right-r.X, bottom-r.Y
	return miniPlayerGeometry(r, container)
}

func (b *Controller) SetMiniPlayerActions(open, restore func()) {
	b.onMiniOpen, b.onMiniRestore = open, restore
}
func (b *Controller) openMiniPlayer(t *Tab) {
	if t == nil || t.Host == nil || !t.Host.Alive() || t.source != nil || t.Document != nil || t.HTML != nil || t.Media != nil {
		return
	}
	b.CancelAction()
	if b.onMiniOpen != nil {
		b.onMiniOpen()
	}
	b.mini = &browserMiniPlayer{tab: t, host: t.Host, geometry: b.miniGeometry, initialized: b.miniGeometry.Width > 0}
	b.update()
}
func (b *Controller) miniPlayerCurrent() bool {
	p := b.mini
	if p == nil || p.tab.Host != p.host || !p.host.Alive() {
		return false
	}
	for _, t := range b.Tabs {
		if t == p.tab {
			return true
		}
	}
	return false
}
func (b *Controller) CloseMiniPlayer() {
	if b.mini == nil {
		return
	}
	b.miniGeometry = b.mini.geometry
	b.mini.host.SetVisible(false)
	b.mini = nil
	b.update()
}
func (b *Controller) restoreMiniPlayer() {
	p := b.mini
	if p == nil {
		return
	}
	b.CloseMiniPlayer()
	for i, t := range b.Tabs {
		if t == p.tab {
			b.Select(i)
			break
		}
	}
	if b.onMiniRestore != nil {
		b.onMiniRestore()
	}
}
func (b *Controller) MiniPlayerBounds() (mygo.Rectangle, bool) {
	if b.mini == nil || !b.mini.rendered {
		return mygo.Rectangle{}, false
	}
	return b.mini.bounds, true
}

// ViewMiniPlayer is called by the conversation overlay after the dock. Its
// bounds are relative to the same parent as the original native browser child.
func (b *Controller) ViewMiniPlayer(c *ui.Context, conversation mygo.Rectangle) {
	if !b.miniPlayerCurrent() {
		b.CloseMiniPlayer()
		return
	}
	p := b.mini
	if conversation.Width < 25 || conversation.Height < 65 {
		p.host.SetVisible(false)
		return
	}
	if !p.initialized {
		p.geometry = miniPlayerGeometry(mygo.Rectangle{X: conversation.Width - 332, Y: 12, Width: 320, Height: 200}, conversation)
		p.initialized = true
	}
	p.geometry = miniPlayerGeometry(p.geometry, conversation)
	r := p.geometry
	p.bounds = mygo.Rectangle{X: conversation.X + r.X, Y: conversation.Y + r.Y, Width: r.Width, Height: r.Height}
	p.rendered = true
	p.context = c
	ui.Overlay(c, func() {
		panel := ui.Column(c).Key(p).Absolute().Left(float32(p.bounds.X)).Top(float32(p.bounds.Y)).Width(float32(r.Width)).Height(float32(r.Height)).Background(c.Theme().Background).Border(1, c.Theme().Border).Clip()
		panel.Children(func() {
			ui.Row(c).Height(32).FillWidth().Padding(4).Gap(4).Children(func() {
				grip := ui.Box(c).Key("browser-mini-move").Grow(1).Height(24).Focusable().Label(browserPreviewText("miniTitle"))
				title := p.host.State().Title
				if title == "" {
					title = p.host.State().URL
				}
				if title == "" {
					title = browserPreviewText("miniTitle")
				}
				grip.Children(func() { ui.Text(c, truncate(title, 32)).FontSize(theme.InterfaceSize(11)).Tooltip(title) })
				dx, dy, held := grip.Dragged()
				if held {
					p.geometry.X += int(dx)
					p.geometry.Y += int(dy)
					p.geometry = miniPlayerGeometry(p.geometry, conversation)
					b.update()
				}
				if ui.Button(c, "↙").Label(browserPreviewText("miniRestore")).Clicked() {
					b.restoreMiniPlayer()
				}
				if ui.Button(c, "×").Label(browserPreviewText("miniClose")).Clicked() {
					b.CloseMiniPlayer()
				}
			})
			for _, side := range []string{"n", "e", "s", "w", "ne", "nw", "se", "sw"} {
				x, y, w, h := 0, 0, r.Width, 5
				cursor := ui.CursorResizeNS
				if side == "s" {
					y = r.Height - 5
				}
				if side == "e" || side == "w" {
					w, h = 5, r.Height
					cursor = ui.CursorResizeEW
					if side == "e" {
						x = r.Width - 5
					}
				}
				if len(side) == 2 {
					w, h = 8, 8
					cursor = ui.CursorResizeNWSE
					if strings.Contains(side, "e") {
						x = r.Width - 8
					}
					if strings.Contains(side, "s") {
						y = r.Height - 8
					}
					if side == "ne" || side == "sw" {
						cursor = ui.CursorResizeNESW
					}
				}
				rail := ui.Box(c).Key("mini-resize-" + side).Absolute().Left(float32(x)).Top(float32(y)).Width(float32(w)).Height(float32(h)).Cursor(cursor).Label(browserChromeLabel("调整小浮窗大小", "Resize mini-window"))
				dx, dy, held := rail.Dragged()
				if held {
					p.geometry = resizeMiniPlayer(p.geometry, int(dx), int(dy), side, conversation)
					b.update()
				}
			}
		})
	})
	if b.mini != p {
		return
	}
	seat := mygo.Rectangle{X: p.bounds.X + 5, Y: p.bounds.Y + 32, Width: max(1, p.bounds.Width-10), Height: max(1, p.bounds.Height-37)}
	p.host.SetViewport(&seat)
	p.host.SetVisible(true)
}
