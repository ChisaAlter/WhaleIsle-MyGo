package pet

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"math"
	"whaleisle/internal/store/storelocale"
	fonttheme "whaleisle/internal/ui/theme"
)

// The footer menu is a native owned popup, rather than an in-card overlay:
// its root/model/effort panes can extend beyond a short quick-chat card.
type quickChatPicker struct {
	chat   *QuickChat
	win    *mygo.Window
	page   string
	height int
}

func (q *QuickChat) closeModelPicker() {
	if p := q.picker; p != nil {
		q.picker = nil
		if p.win != nil && !p.win.IsDestroyed() {
			p.win.Destroy()
		}
		q.publishChatBounds()
	}
}
func (q *QuickChat) openModelPicker() {
	if !q.check() {
		return
	}
	q.closeModelPicker()
	p := &quickChatPicker{chat: q, page: "root"}
	q.picker = p
	w := mygo.NewWindow(mygo.WindowOptions{Title: storelocale.T(storelocale.KeyDesktopPetSelectSharedModel), Width: 232, Height: 90, Parent: q.win, AlwaysOnTop: true, Frameless: true, SkipTaskbar: true, DisableResize: true, DisableMinimize: true, DisableMaximize: true, DisableFullScreen: true, Content: ui.View(p.view)})
	p.win = w
	if w == nil || w.IsDestroyed() {
		q.picker = nil
		return
	}
	w.OnBlur(func() {
		if q.picker == p {
			q.closeModelPicker()
		}
	})
	w.OnClosed(func() {
		if q.picker == p {
			q.picker = nil
			q.publishChatBounds()
		}
	})
	p.position()
	w.Focus()
}
func (p *quickChatPicker) position() {
	q := p.chat
	if p.win == nil || p.win.IsDestroyed() || q.win == nil || q.win.IsDestroyed() {
		return
	}
	card := petWindowBounds(q.win)
	scale := petCoordinateScale(activePetDisplay(petDisplays(), q.owner.petBounds()))
	width, height := int(math.Round(232*scale)), int(math.Round(float64(max(90, p.height))*scale))
	x, y := card.X, card.Y+card.Height-int(34*scale)-height
	if a, ok := motionWorkArea(q.owner.petBounds()); ok {
		left, top := int(math.Round(a.x)), int(math.Round(a.y))
		right, bottom := int(math.Round(a.x+a.width)), int(math.Round(a.y+a.height))
		x = min(max(x, left+4), max(left+4, right-width-4))
		y = min(max(y, top+4), max(top+4, bottom-height-4))
	}
	petPlaceNative(p.win, mygo.Rectangle{X: x, Y: y, Width: width, Height: height})
	q.publishChatBounds()
}
func (p *quickChatPicker) view(c *ui.Context) {
	q := p.chat
	q.applyTheme(c)
	t := c.Theme()
	selected := q.state.Model
	blocked := !q.check() || q.sending.busy() || q.picking.busy() || q.stopping.busy()
	count := 2
	if p.page == "model" {
		count = 1
		for _, g := range q.state.Groups {
			count += len(g.Models) + 1
		}
	}
	if p.page == "effort" {
		count = 2
		for _, g := range q.state.Groups {
			for _, m := range g.Models {
				if selected != nil && selected.Provider == g.ID && selected.Model == m.ID {
					count += len(m.Efforts)
				}
			}
		}
	}
	height := max(90, min(320, 12+count*30))
	if p.win != nil && p.height != height {
		p.height = height
		p.win.SetSize(232, height)
		p.position()
	}
	ui.Column(c).Fill().Padding(6).Gap(2).Background(t.Background).Border(1, t.Border).Radius(10).Children(func() {
		action := func(label string, disabled, checked bool, run func()) {
			if checked {
				label = "✓ " + label
			}
			if ui.Button(c, label).FillWidth().MinWidth(0).Height(28).Padding(4, 8).BorderWidth(0).Background(ui.Transparent).FontSize(fonttheme.InterfaceSize(11)).TextColor(t.Text).SingleLine().Tooltip(label).Disabled(disabled).Clicked() {
				run()
			}
		}
		if p.page == "root" {
			action(storelocale.T(storelocale.KeyDesktopPetModel)+"  ›", blocked, false, func() { p.page = "model" })
			hasEffort := false
			for _, g := range q.state.Groups {
				for _, m := range g.Models {
					if selected != nil && g.ID == selected.Provider && m.ID == selected.Model && len(m.Efforts) > 0 {
						hasEffort = true
					}
				}
			}
			action(storelocale.T(storelocale.KeyDesktopPetReasoning)+"  ›", blocked || !hasEffort, false, func() { p.page = "effort" })
		} else {
			action("‹ "+quickChatPickerCopy("返回", "Back"), false, false, func() { p.page = "root" })
			ui.Scroll(c).FillWidth().MinHeight(0).Grow(1).Children(func() {
				if p.page == "model" {
					for _, g := range q.state.Groups {
						ui.Text(c, quickChatName(g.Name, g.ID)).FillWidth().FontSize(fonttheme.InterfaceSize(10)).TextColor(t.TextMuted).Padding(7, 8, 3, 8)
						for _, m := range g.Models {
							choice := quickChatModel{Provider: g.ID, Model: m.ID, ReasoningEffort: m.DefaultEffort}
							if !q.state.allows(choice) {
								choice.ReasoningEffort = ""
							}
							picked := selected != nil && selected.Provider == g.ID && selected.Model == m.ID
							action(quickChatName(m.Name, m.ID), blocked, picked, func() { q.closeModelPicker(); q.choose(choice) })
						}
					}
				} else if selected != nil {
					action(storelocale.T(storelocale.KeyDesktopPetDefault), blocked, selected.ReasoningEffort == "", func() { choice := *selected; choice.ReasoningEffort = ""; q.closeModelPicker(); q.choose(choice) })
					for _, g := range q.state.Groups {
						for _, m := range g.Models {
							if g.ID == selected.Provider && m.ID == selected.Model {
								for _, e := range m.Efforts {
									choice := *selected
									choice.ReasoningEffort = e.ID
									action(quickChatName(e.Name, e.ID), blocked, selected.ReasoningEffort == e.ID, func() { q.closeModelPicker(); q.choose(choice) })
								}
							}
						}
					}
				}
			})
		}
	})
	if c.Shortcut(0, ui.KeyEscape) {
		if p.page != "root" {
			p.page = "root"
		} else {
			q.closeModelPicker()
		}
	}
}
func quickChatPickerCopy(zh, en string) string {
	if storelocale.Locale() == "en" {
		return en
	}
	return zh
}
