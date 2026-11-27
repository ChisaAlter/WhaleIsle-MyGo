package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TodoPanelModel drives Electron TodoPanel (composer dock strip — todos only).
type TodoPanelModel struct {
	Todos      []session.PlanTodo
	ThemeID    string
	SystemDark bool
	// Collapsed nil uses package fallback (default true = Electron).
	Collapsed         *bool
	PendingSubmission bool
	PlanSignature     string
}

var todoPanelCollapsedFB = false

var (
	// Delicate outline list-bullet icon and compact chevron matching Electron TodoPanel header
	todoChecklistIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.05" stroke-linecap="round" stroke-linejoin="round"><circle cx="3.5" cy="4.75" r="1.55"/><path d="M7.25 4.75H13.5"/><circle cx="3.5" cy="11.25" r="1.55"/><path d="M7.25 11.25H13.5"/></svg>`))
	todoChevronDown   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.05" stroke-linecap="round" stroke-linejoin="round"><path d="M5 6.5L8 9.5L11 6.5"/></svg>`))
	todoChevronUp     = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.05" stroke-linecap="round" stroke-linejoin="round"><path d="M5 9.5L8 6.5L11 9.5"/></svg>`))
)

type todoDockState struct {
	leaving   PlanLeavingTracker
	lastTodos []session.PlanTodo
	contentH  float32
}

// TodoPanelView paints checklist header + optional expanded todo rows.
// Animates via 200ms clipped vertical slide, and hides immediately on pending
// submission (usePlanLeavingOnSend) without blocking composer motion.
func TodoPanelView(c *ui.Context, m TodoPanelModel) {
	sig := m.PlanSignature
	if sig == "" && len(m.Todos) > 0 {
		sig = fmt.Sprintf("%d:%s:%s", len(m.Todos), m.Todos[0].Content, m.Todos[0].Status)
	}
	host := ui.Column(c.Key("todo-panel-dock")).FillWidth()
	st := ui.Local(host, "todo-dock-state", func() todoDockState { return todoDockState{} })
	now := c.Now()
	leaving := st.leaving.Update(now, m.PendingSubmission, sig)
	if leaving && !now.IsZero() {
		if !st.leaving.GraceUntil.IsZero() {
			if rem := st.leaving.GraceUntil.Sub(now); rem > 0 {
				c.After(rem)
			}
		} else if !st.leaving.PendingSince.IsZero() {
			if rem := st.leaving.PendingSince.Add(PlanLeavingMaxDuration).Sub(now); rem > 0 {
				c.After(rem)
			}
		}
	}
	open := len(m.Todos) > 0 && !leaving
	if open {
		st.lastTodos = m.Todos
	}
	progress := DockSlideProgress(c, host, open, leaving)
	if !open && progress <= 0 {
		st.lastTodos = nil
		return
	}
	todos := m.Todos
	if !open {
		todos = st.lastTodos
	}
	if len(todos) == 0 {
		return
	}
	t := design.For(c)
	fg := t.Foreground
	muted := t.MutedForeground
	accent := t.Primary
	ok := t.SuccessForeground
	labelSecondary := fg.Alpha(0.88)

	collapsed := &todoPanelCollapsedFB
	if m.Collapsed != nil {
		collapsed = m.Collapsed
	}
	allDone := len(todos) > 0
	for _, todo := range todos {
		switch strings.ToLower(strings.TrimSpace(todo.Status)) {
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

	paintDockSlide(c, host, progress, open, &st.contentH, func(contentOpacity float32) {
		// TaskDock.module.css + TodoPanel.module.css: 32px side inset inside composerOuterW (W+64) so TodoPanel width equals W (680px).
		ui.Column(c).FillWidth().Padding(0, design.Sp(8), design.Sp(1.25), design.Sp(8)).Children(func() {
			card := chatCardSurface(c, ui.Column(c).FillWidth().Padding(design.Sp(1.5), design.Sp(3), design.Sp(1.5), design.Sp(3)).Gap(0), design.RadiusLG).Clip()
			if card.Hovered() {
				card.Background(t.Accent)
			}
			card.Children(func() {
				content := ui.Column(c).FillWidth().Opacity(contentOpacity)
				content.Children(func() {
					hdr := coss.Button(c, "", coss.ButtonOptions{
						Variant: coss.ButtonGhost,
						Size:    coss.ButtonXS,
						Label:   i18n.T(i18n.KeyTodoPanelTitle),
						Content: func() {
							ui.Row(c).FillWidth().Gap(design.Sp(2.5)).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, todoChecklistIcon).Size(16, 16).TextColor(muted.Alpha(0.85)).PassThrough()
								ui.Text(c, i18n.T(i18n.KeyTodoPanelTitle)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(24).FontWeight(500).TextColor(fg).PassThrough()
								ui.Text(c, todoProgressLabel(todos)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).FontWeight(400).TextColor(muted).Grow(1).PassThrough()
								chev := todoChevronDown
								if *collapsed {
									chev = todoChevronUp
								}
								ui.Icon(c, chev).Size(16, 16).TextColor(muted.Alpha(0.82)).PassThrough()
							})
						},
					}).FillWidth().Padding(0)
					if hdr.Hovered() {
						card.Background(t.Accent)
					}
					if hdr.Clicked() {
						*collapsed = !*collapsed
					}
					if *collapsed {
						return
					}
					ui.Column(c).FillWidth().Padding(design.Sp(2), 0, 0, 0).Gap(design.Sp(2)).MaxHeight(180).Children(func() {
						for _, todo := range todos {
							stColor := muted
							switch strings.ToLower(strings.TrimSpace(todo.Status)) {
							case "completed", "complete", "done":
								stColor = ok
							case "in_progress", "running", "active":
								stColor = accent
							}
							ui.Row(c).FillWidth().Gap(design.Sp(2.5)).AlignItems(ui.Center).Children(func() {
								ui.Box(c).Size(16, 16).Shrink(0).Center().Children(func() {
									ui.Box(c).Size(6, 6).Radius(design.RadiusFull).Background(stColor)
								})
								ui.Text(c, todo.Content).FontSize(theme.InterfaceSize(13)).FixedLineHeight(20).TextColor(labelSecondary).Grow(1)
							})
						}
					})
				})
			})
		})
	})
}

func todoCompletedFraction(todos []session.PlanTodo) string {
	if len(todos) == 0 {
		return ""
	}
	done := 0
	for _, t := range todos {
		switch strings.ToLower(strings.TrimSpace(t.Status)) {
		case "completed", "complete", "done":
			done++
		}
	}
	return todoCompletedFractionCount(done, len(todos))
}

func todoCompletedFractionCount(done, total int) string {
	return fmt.Sprintf("%d/%d %s", done, total, historyNavigationText("已完成", "completed"))
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
	if len(todos) > 0 {
		parts = append(parts, todoCompletedFractionCount(done, len(todos)))
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
