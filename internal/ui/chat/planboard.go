package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// PlanBoardModel drives the compact plan/goal/queue/todo panel under chat.
type PlanBoardModel struct {
	QueueEditor       session.QueueEditorState
	OnChangeQueueEdit func(token uint64, text string)
	OnSaveQueueEdit   func(token uint64)
	OnCancelQueueEdit func(token uint64)
	Board             session.PlanBoard
	LoadQueueImage    func(context.Context, string) (string, error)
	ThemeID           string
	SystemDark        bool
	// HideQueue moves the queue rows and the inline queue editor to the
	// composer dock (QueueDockView); the goal, plan and todo sections stay here.
	HideQueue bool
	HideGoal  bool
	// Open nil means always expanded; non-nil is a toggle.
	Open *bool
	// GoalDraft is the create/edit objective field (Electron goals strip).
	GoalDraft *string
	// OnCreateGoal maps to goals/create.
	OnCreateGoal func(objective string)
	// OnEditGoal maps to goals/edit.
	OnEditGoal func(objective string)
	// OnCompleteGoal maps to goals/complete.
	OnCompleteGoal func()
	// OnClearGoal maps to goals/clear.
	OnClearGoal func()
	// OnPauseGoal maps to goals/pause.
	OnPauseGoal func()
	// OnResumeGoal maps to goals/resume.
	OnResumeGoal func()
	// OnQueueRemove maps to session/updateQueue action=remove.
	OnQueueRemove func(itemID string)
	// OnQueueSteer maps to session/updateQueue action=steer.
	OnQueueSteer func(itemID string)
	// OnQueueEdit prefills composer / submits edit action with preview text.
	OnQueueEdit func(itemID, preview string)
}

// PlanBoardView paints the M5 plan strip (empty state OK).
func PlanBoardView(c *ui.Context, m PlanBoardModel) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	accent := chrome.Color(colors.Accent)
	cardBG := chrome.Color(colors.Field)
	dark := th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark) || darkSurface(fg)

	ok := ui.RGB(22, 163, 74)
	panelBG := ui.RGB(248, 249, 250)
	subtleBorder := ui.RGBA(0, 0, 0, 0.06)
	if dark {
		ok = ui.RGB(34, 197, 94)
		panelBG = ui.RGB(35, 35, 36)
		subtleBorder = ui.RGBA(255, 255, 255, 0.08)
	}

	open := true
	if m.Open != nil {
		open = *m.Open
	}
	// Collapsed: thin expand strip (Electron D-PLAN affordance) so the board
	// can be reopened; expanded paints the full plan/goal/queue/todo panel.
	if m.Open != nil && !open {
		chrome.GlassSurface(ui.Column(c).FillWidth().MinWidth(0), panelBG, chrome.MaterialPanel, 12).
			Border(0.5, subtleBorder).Children(func() {
			ui.Row(c).Padding(8, 12, 8, 12).Gap(8).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyPlanBoardTitle)).FontSize(theme.InterfaceSize(12.5)).Bold().TextColor(fg)
				sum := planBoardSummary(m.Board)
				if sum != "" {
					ui.Text(c, sum).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1)
				} else {
					ui.Box(c).Grow(1).MinWidth(0)
				}
				if ui.Button(c, i18n.T(i18n.KeyPlanBoardExpand)).FontSize(theme.InterfaceSize(12)).
					Height(24).Padding(0, 8, 0, 8).Radius(6).
					BorderWidth(0).Background(ui.Transparent).TextColor(accent).Clicked() {
					*m.Open = true
				}
			})
		})
		return
	}

	chrome.GlassSurface(ui.Column(c).FillWidth().MinWidth(0), panelBG, chrome.MaterialPanel, 12).
		Border(0.5, subtleBorder).Children(func() {
		hdr := ui.Row(c).Padding(8, 12, 8, 12).Gap(8).FillWidth().AlignItems(ui.Center)
		hdr.Children(func() {
			ui.Text(c, i18n.T(i18n.KeyPlanBoardTitle)).FontSize(theme.InterfaceSize(12.5)).Bold().TextColor(fg).Grow(1)
			ui.Text(c, planBoardSummary(m.Board)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if m.Open != nil {
				if ui.Button(c, i18n.T(i18n.KeyPlanBoardCollapse)).FontSize(theme.InterfaceSize(12)).
					Height(24).Padding(0, 8, 0, 8).Radius(6).
					BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
					*m.Open = false
				}
			}
		})
		ui.Box(c).FillWidth().Height(0.5).Background(subtleBorder)
		ui.Column(c).FillWidth().MinWidth(0).Padding(8, 12, 12, 12).Gap(8).Children(func() {
			if m.Board.Empty() {
				ui.Text(c, i18n.T(i18n.KeyPlanBoardEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			} else {
				paintPlanSection(c, i18n.T(i18n.KeyPlanBoardPlan), muted, fg, func() {
					label := i18n.T(i18n.KeyPlanBoardPlanOff)
					color := muted
					if m.Board.PlanMode {
						label = i18n.T(i18n.KeyPlanBoardPlanOn)
						color = accent
					}
					ui.Text(c, label).FontSize(theme.InterfaceSize(12.5)).TextColor(color)
				})
			}
			if !m.HideGoal {
				paintPlanSection(c, i18n.T(i18n.KeyPlanBoardGoal), muted, fg, func() {
					paintPlanGoal(c, m, fg, muted, accent, subtleBorder, cardBG)
				})
			}
			if !m.HideQueue {
				paintQueueEditor(c, m, fg, muted)
			}
			if m.Board.Empty() {
				return
			}
			if !m.HideQueue {
				paintPlanSection(c, i18n.T(i18n.KeyPlanBoardQueue), muted, fg, func() {
					for _, pending := range m.Board.Pending {
						paintPendingSubmission(c, pending, fg, muted)
					}
					if len(m.Board.Queue) == 0 && len(m.Board.Pending) == 0 {
						ui.Text(c, i18n.T(i18n.KeyPlanBoardQueueEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
						return
					}
					for _, q := range m.Board.Queue {
						item := q
						line := item.Preview
						if item.Source != "" {
							line = item.Source + ": " + line
						}
						ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(6, 8, 6, 8).Radius(8).
							Border(0.5, subtleBorder).Background(cardBG).AlignItems(ui.Center).Children(func() {
							ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
								paintQueueAttachments(c, m, item, fg, muted)
								ui.Text(c, maskedUserText(line)).FontSize(theme.InterfaceSize(12.5)).TextColor(fg)
							})
							if m.OnQueueEdit != nil && item.ID != "" {
								if ui.Button(c, i18n.T(i18n.KeyPlanBoardQueueEdit)).Disabled(!m.Board.QueueActions.Can(session.QueueActionEdit, item)).FontSize(theme.InterfaceSize(11)).
									Height(24).Padding(0, 6, 0, 6).Radius(6).BorderWidth(0).
									Background(ui.Transparent).TextColor(muted).Clicked() {
									m.OnQueueEdit(item.ID, item.Preview)
								}
							}
							if m.OnQueueSteer != nil && item.ID != "" {
								if ui.Button(c, i18n.T(i18n.KeyPlanBoardQueueSteer)).Disabled(!m.Board.QueueActions.Can(session.QueueActionSteer, item)).FontSize(theme.InterfaceSize(11)).
									Height(24).Padding(0, 6, 0, 6).Radius(6).BorderWidth(0).
									Background(ui.Transparent).TextColor(accent).Clicked() {
									m.OnQueueSteer(item.ID)
								}
							}
							if m.OnQueueRemove != nil && item.ID != "" {
								if ui.Button(c, i18n.T(i18n.KeyPlanBoardQueueRemove)).Disabled(!m.Board.QueueActions.Can(session.QueueActionRemove, item)).FontSize(theme.InterfaceSize(11)).
									Height(24).Padding(0, 6, 0, 6).Radius(6).BorderWidth(0).
									Background(ui.Transparent).TextColor(muted).Clicked() {
									m.OnQueueRemove(item.ID)
								}
							}
						})
					}
				})
			}
			paintPlanSection(c, i18n.T(i18n.KeyPlanBoardTodos), muted, fg, func() {
				if len(m.Board.Todos) == 0 {
					ui.Text(c, i18n.T(i18n.KeyPlanBoardTodosEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					return
				}
				ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
					for _, todo := range m.Board.Todos {
						stColor := muted
						textColor := fg
						switch strings.ToLower(todo.Status) {
						case "completed", "complete", "done":
							stColor = ok
							textColor = muted
						case "in_progress", "running", "active":
							stColor = accent
						case "cancelled", "canceled":
							textColor = muted
						}
						ui.Row(c).Gap(8).Padding(3, 2, 3, 2).FillWidth().AlignItems(ui.Center).Children(func() {
							ui.Text(c, todoStatusGlyph(todo.Status)).FontSize(theme.InterfaceSize(12)).TextColor(stColor)
							ui.Text(c, todo.Content).FontSize(theme.InterfaceSize(12.5)).TextColor(textColor).Grow(1)
							ui.Text(c, todo.Status).FontSize(theme.InterfaceSize(11)).TextColor(stColor)
						})
					}
				})
			})
		})
	})
}

func paintPlanGoal(c *ui.Context, m PlanBoardModel, fg, muted, accent, subtleBorder, cardBG ui.Color) {
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	if m.Board.Goal == nil {
		ui.Text(c, i18n.T(i18n.KeyPlanBoardNoGoal)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if m.GoalDraft != nil && m.OnCreateGoal != nil {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, m.GoalDraft).Grow(1).
					Placeholder(i18n.T(i18n.KeyPlanBoardGoalDraftPlaceholder))
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalCreate), buttonGhost, false).
					BorderWidth(0).Background(ui.Transparent).TextColor(accent).Clicked() {
					obj := strings.TrimSpace(*m.GoalDraft)
					if obj != "" {
						m.OnCreateGoal(obj)
						*m.GoalDraft = ""
					}
				}
			})
		}
		return
	}
	g := m.Board.Goal
	phase := g.Phase
	if phase == "" {
		phase = "?"
	}
	status := phase
	if g.Activation != "" {
		status += " / " + g.Activation
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(6).Padding(8, 10, 8, 10).Radius(8).
		Border(0.5, subtleBorder).Background(cardBG).Children(func() {
		ui.Text(c, fmt.Sprintf("[%s] %s", status, g.Objective)).FontSize(theme.InterfaceSize(12.5)).TextColor(fg)
		if g.BlockedReason.Code != "" || g.BlockedReason.Message != "" {
			reason := g.BlockedReason.Message
			if g.BlockedReason.Code != "" {
				reason = "[" + g.BlockedReason.Code + "] " + reason
			}
			ui.Text(c, reason).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if m.GoalDraft != nil && m.OnEditGoal != nil {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, m.GoalDraft).Grow(1).
					Placeholder(i18n.T(i18n.KeyPlanBoardGoalDraftPlaceholder))
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalEdit), buttonGhost, false).
					BorderWidth(0).Background(ui.Transparent).TextColor(accent).Clicked() {
					obj := strings.TrimSpace(*m.GoalDraft)
					if obj != "" {
						m.OnEditGoal(obj)
					}
				}
			})
		}
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			resume := g.Phase == "paused" || (g.Phase == "active" && g.Activation == "disarmed")
			if resume {
				if m.OnResumeGoal != nil {
					if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalResume), buttonGhost, false).
						BorderWidth(0).Background(ui.Transparent).TextColor(accent).Clicked() {
						m.OnResumeGoal()
					}
				}
			} else if g.Phase == "active" && g.Activation == "armed" && m.OnPauseGoal != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalPause), buttonGhost, false).
					BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
					m.OnPauseGoal()
				}
			}
			if m.OnCompleteGoal != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalComplete), buttonPrimary, false).
					BorderWidth(0).Background(ui.Transparent).TextColor(accent).Clicked() {
					m.OnCompleteGoal()
				}
			}
			if m.OnClearGoal != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalClear), buttonDanger, false).
					BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
					m.OnClearGoal()
				}
			}
		})
	})
}

func paintPlanSection(c *ui.Context, title string, muted, fg ui.Color, body func()) {
	_ = fg
	ui.Column(c).Gap(6).FillWidth().MinWidth(0).Children(func() {
		ui.Text(c, title).FontSize(theme.InterfaceSize(11)).Bold().TextColor(muted)
		body()
	})
}

// PlanBoardSummary is a one-line status for the compact progress strip.
func PlanBoardSummary(b session.PlanBoard) string {
	return planBoardSummary(b)
}

func planBoardSummary(b session.PlanBoard) string {
	if b.Empty() {
		return ""
	}
	// Electron TodoPanel progressLabel: status counts joined with en-space · .
	if len(b.Todos) > 0 {
		done, active, pending := 0, 0, 0
		for _, t := range b.Todos {
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
	parts := make([]string, 0, 3)
	if b.PlanMode {
		parts = append(parts, i18n.T(i18n.KeyPlanBoardPlanOn))
	}
	if b.Goal != nil {
		parts = append(parts, i18n.T(i18n.KeyPlanBoardGoal))
	}
	if n := len(b.Queue) + len(b.Pending); n > 0 {
		parts = append(parts, fmt.Sprintf("%s %d", i18n.T(i18n.KeyPlanBoardQueue), n))
	}
	return strings.Join(parts, "\u2002·\u2002")
}

func todoStatusGlyph(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "done":
		return "✓"
	case "in_progress", "running", "active":
		return "●"
	case "cancelled", "canceled":
		return "×"
	default:
		return "○"
	}
}

func paintQueueEditor(c *ui.Context, m PlanBoardModel, fg, muted ui.Color) {
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	e := m.QueueEditor
	if e.ItemID == "" {
		return
	}
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyPlanBoardQueueEdit)).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		if e.Ready {
			draft := e.Text
			ui.TextArea(c.Key("queue-edit-"+e.ItemID), &draft).FillWidth().Lines(2, 6).ReadOnly(e.Saving).OnChange(func() {
				if m.OnChangeQueueEdit != nil {
					m.OnChangeQueueEdit(e.Token, draft)
				}
			})
		}
		if e.Error != "" {
			ui.Text(c, e.Error).FontSize(theme.InterfaceSize(11)).TextColor(fg)
		}
		ui.Row(c).Gap(8).Children(func() {
			if e.Ready && m.OnSaveQueueEdit != nil && ui.Button(c, i18n.T(i18n.KeySettingsSave)).Disabled(e.Saving || strings.TrimSpace(e.Text) == "").Clicked() {
				m.OnSaveQueueEdit(e.Token)
			}
			if m.OnCancelQueueEdit != nil && stdButton(c, palette, i18n.T(i18n.KeySidebarRenameCancel), buttonGhost, e.Saving).Clicked() {
				m.OnCancelQueueEdit(e.Token)
			}
		})
	})
}
