package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TodoPanelModel drives Electron TodoPanel (composer dock strip — todos only).
type TodoPanelModel struct {
	Todos      []session.PlanTodo
	ThemeID    string
	SystemDark bool
	// Collapsed nil uses package fallback (default true = Electron).
	Collapsed *bool
}

var todoPanelCollapsedFB = false

var (
	// Delicate outline list-bullet icon and compact chevron matching Electron TodoPanel header
	todoChecklistIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"><circle cx="3.5" cy="4.75" r="1.55"/><path d="M7.25 4.75H13.5"/><circle cx="3.5" cy="11.25" r="1.55"/><path d="M7.25 11.25H13.5"/></svg>`))
	todoChevronDown   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.15" stroke-linecap="round" stroke-linejoin="round"><path d="M5 6.5L8 9.5L11 6.5"/></svg>`))
	todoChevronUp     = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.15" stroke-linecap="round" stroke-linejoin="round"><path d="M5 9.5L8 6.5L11 9.5"/></svg>`))
)

// TodoPanelView paints checklist header + optional expanded todo rows.
// Returns without painting when Todos is empty (Electron null render).
func TodoPanelView(c *ui.Context, m TodoPanelModel) {
	if len(m.Todos) == 0 {
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	accent := chrome.Color(colors.Accent)
	dark := th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark)

	ok := ui.RGB(22, 163, 74)
	labelSecondary := fg.Alpha(0.88)
	tipBG := ui.RGBA(255, 255, 255, 0.62)
	tipHoverBG := ui.RGBA(255, 255, 255, 0.74)
	border := ui.RGBA(0, 0, 0, 0.04)
	if dark {
		ok = ui.RGB(34, 197, 94)
		labelSecondary = ui.RGB(207, 211, 214)
		muted = ui.RGB(173, 178, 184)
		tipBG = ui.RGBA(215, 230, 250, 0.055)
		tipHoverBG = ui.RGBA(215, 230, 250, 0.08)
		border = ui.RGBA(255, 255, 255, 0.03)
	}

	collapsed := &todoPanelCollapsedFB
	if m.Collapsed != nil {
		collapsed = m.Collapsed
	}
	allDone := len(m.Todos) > 0
	for _, t := range m.Todos {
		switch strings.ToLower(strings.TrimSpace(t.Status)) {
		case "completed", "complete", "done":
		default:
			allDone = false
		}
	}
	doneSeat := ui.Box(c.Key("todo-panel-done-seat")).Width(0).Height(0)
	autoCollapsedDone := ui.Local(doneSeat, "auto-collapsed-done", func() bool { return false })
	if allDone {
		if !*autoCollapsedDone {
			*autoCollapsedDone = true
			*collapsed = true
		}
	} else {
		*autoCollapsedDone = false
	}

	// TaskDock.module.css + TodoPanel.module.css: 32px side inset inside composerOuterW (W+64) so TodoPanel width equals W (680px).
	ui.Column(c).FillWidth().Padding(0, 32, 5, 32).Children(func() {
		card := ui.Column(c).FillWidth().Radius(12).Background(tipBG).
			Padding(6, 12, 6, 12).Gap(0).Clip().DrawOver(func(p *ui.Painter, r ui.Rect) {
			if border.A > 0 && r.W > 0 && r.H > 0 {
				p.Stroke(r, border, 12, 0.5)
			}
		})
		if card.Hovered() {
			card.Background(tipHoverBG)
		}
		card.Children(func() {
			hdr := ui.Button(c, "").FillWidth().
				Background(ui.Transparent).
				BorderWidth(0).Padding(0, 0, 0, 0)
			if hdr.Hovered() {
				card.Background(tipHoverBG)
			}
			hdr.Children(func() {
				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, todoChecklistIcon).Size(16, 16).TextColor(muted.Alpha(0.85)).PassThrough()
					ui.Text(c, i18n.T(i18n.KeyTodoPanelTitle)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(24).FontWeight(500).TextColor(fg).PassThrough()
					ui.Text(c, todoProgressLabel(m.Todos)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).FontWeight(400).TextColor(muted).Grow(1).PassThrough()
					chev := todoChevronDown
					if *collapsed {
						chev = todoChevronUp
					}
					ui.Icon(c, chev).Size(16, 16).TextColor(muted.Alpha(0.82)).PassThrough()
				})
			})
			if hdr.Clicked() {
				*collapsed = !*collapsed
			}
			if *collapsed {
				return
			}
			ui.Column(c).FillWidth().Padding(8, 0, 0, 0).Gap(8).MaxHeight(180).Children(func() {
				for _, todo := range m.Todos {
					stColor := muted
					switch strings.ToLower(strings.TrimSpace(todo.Status)) {
					case "completed", "complete", "done":
						stColor = ok
					case "in_progress", "running", "active":
						stColor = accent
					}
					ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(16, 16).Shrink(0).Center().Children(func() {
							ui.Box(c).Size(6, 6).Radius(3).Background(stColor)
						})
						ui.Text(c, todo.Content).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).TextColor(labelSecondary).Grow(1)
					})
				}
			})
		})
	})
}

func todoProgressLabel(todos []session.PlanTodo) string {
	done, active, pending := 0, 0, 0
	for _, t := range todos {
		switch strings.ToLower(strings.TrimSpace(t.Status)) {
		case "completed", "complete", "done":
			done++
		case "in_progress", "running", "active":
			active++
		default:
			pending++
		}
	}
	parts := make([]string, 0, 3)
	if done > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", done, historyNavigationText("已完成", "completed")))
	}
	if active > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", active, historyNavigationText("进行中", "in progress")))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", pending, historyNavigationText("待处理", "pending")))
	}
	return strings.Join(parts, "\u2002·\u2002")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
