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

var todoPanelCollapsedFB = true

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
	ok := ui.RGB(61, 153, 112)

	collapsed := &todoPanelCollapsedFB
	if m.Collapsed != nil {
		collapsed = m.Collapsed
	}

	// TodoPanel.module.css: tip surface, border-l1, radius 12, body pad 6/12.
	tipBG := chrome.Color("rgba(255, 255, 255, 0.72)")
	border := chrome.Color(colors.Border)
	if th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark) {
		tipBG = chrome.Color(colors.Field)
	}
	ui.Column(c).FillWidth().Padding(0, 16, 8, 16).Children(func() {
		card := ui.Column(c).FillWidth().Background(tipBG).Border(1, border).Radius(12).
			Padding(6, 12, 6, 12).Gap(8)
		card.Children(func() {
			hdr := ui.Button(c, "").FillWidth().
				Background(ui.Transparent).
				BorderWidth(0).Padding(0, 0, 0, 0)
			hdr.Children(func() {
				ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "☑").FontSize(theme.InterfaceSize(13)).TextColor(muted)
					ui.Text(c, i18n.T(i18n.KeyTodoPanelTitle)).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg)
					ui.Text(c, todoProgressLabel(m.Todos)).FontSize(theme.InterfaceSize(13)).TextColor(muted).Grow(1)
					chev := "⌃"
					if *collapsed {
						chev = "⌄"
					}
					ui.Text(c, chev).FontSize(theme.InterfaceSize(13)).TextColor(muted)
				})
			})
			if hdr.Clicked() {
				*collapsed = !*collapsed
			}
			if *collapsed {
				return
			}
			ui.Column(c).FillWidth().Gap(8).MaxHeight(180).Children(func() {
				for _, todo := range m.Todos {
					stColor := muted
					switch strings.ToLower(strings.TrimSpace(todo.Status)) {
					case "completed", "complete", "done":
						stColor = ok
					case "in_progress", "running", "active":
						stColor = accent
					}
					ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, todoStatusGlyph(todo.Status)).FontSize(theme.InterfaceSize(12)).TextColor(stColor)
						ui.Text(c, todo.Content).FontSize(theme.InterfaceSize(13)).TextColor(fg).Grow(1)
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
		parts = append(parts, fmt.Sprintf(i18n.T(i18n.KeyPlanBoardTodoDone), done))
	}
	if active > 0 {
		parts = append(parts, fmt.Sprintf(i18n.T(i18n.KeyPlanBoardTodoActive), active))
	}
	if pending > 0 {
		parts = append(parts, fmt.Sprintf(i18n.T(i18n.KeyPlanBoardTodoPending), pending))
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
