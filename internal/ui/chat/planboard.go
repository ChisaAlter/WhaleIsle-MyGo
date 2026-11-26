package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
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
	// OnQueueEdit prefills composer / submits edit action with preview text.
	OnQueueEdit func(itemID, preview string)
	// OnQueueMove maps to session/updateQueue action=move.
	OnQueueMove       func(itemID string, toIndex int)
	OnQueueMoveResult func(itemID string, toIndex int) bool
	QueueMoveFailed   bool
	QueueMoveError    string
	PendingSubmission bool
	UserQueueAction   bool
}

// PlanBoardView paints the M5 plan strip (empty state OK).
func PlanBoardView(c *ui.Context, m PlanBoardModel) {
	t := design.For(c)
	fg := t.Foreground
	muted := t.MutedForeground
	accent := t.Primary
	cardBG := t.Muted
	ok := t.SuccessForeground
	subtleBorder := t.Border

	open := true
	if m.Open != nil {
		open = *m.Open
	}
	// Collapsed: thin expand strip (Electron D-PLAN affordance) so the board
	// can be reopened; expanded paints the full plan/goal/queue/todo panel.
	if m.Open != nil && !open {
		chatCardSurface(c, ui.Column(c).FillWidth().MinWidth(0), design.RadiusLG).Children(func() {
			ui.Row(c).Padding(design.Sp(2), design.Sp(3), design.Sp(2), design.Sp(3)).Gap(design.Sp(2)).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyPlanBoardTitle)).FontSize(theme.InterfaceSize(12.5)).Bold().TextColor(fg)
				sum := planBoardSummary(m.Board)
				if sum != "" {
					ui.Text(c, sum).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1)
				} else {
					ui.Box(c).Grow(1).MinWidth(0)
				}
				if coss.Button(c, i18n.T(i18n.KeyPlanBoardExpand), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
					*m.Open = true
				}
			})
		})
		return
	}

	chatCardSurface(c, ui.Column(c).FillWidth().MinWidth(0), design.RadiusLG).Children(func() {
		hdr := ui.Row(c).Padding(design.Sp(2), design.Sp(3), design.Sp(2), design.Sp(3)).Gap(design.Sp(2)).FillWidth().AlignItems(ui.Center)
		hdr.Children(func() {
			ui.Text(c, i18n.T(i18n.KeyPlanBoardTitle)).FontSize(theme.InterfaceSize(12.5)).Bold().TextColor(fg).Grow(1)
			ui.Text(c, planBoardSummary(m.Board)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if m.Open != nil {
				if coss.Button(c, i18n.T(i18n.KeyPlanBoardCollapse), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
					*m.Open = false
				}
			}
		})
		coss.Separator(c, coss.Horizontal)
		ui.Column(c).FillWidth().MinWidth(0).Padding(design.Sp(2), design.Sp(3), design.Sp(3), design.Sp(3)).Gap(design.Sp(2)).Children(func() {
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
						ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).Padding(design.Sp(1.5), design.Sp(2), design.Sp(1.5), design.Sp(2)).Radius(design.RadiusMD).
							Border(design.BorderWidth, subtleBorder).Background(cardBG).AlignItems(ui.Center).Children(func() {
							ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(1)).Children(func() {
								paintQueueAttachments(c, m, item, fg, muted)
								ui.Text(c, maskedUserText(line)).FontSize(theme.InterfaceSize(12.5)).TextColor(fg)
							})
							if m.OnQueueEdit != nil && item.ID != "" {
								if coss.Button(c, i18n.T(i18n.KeyPlanBoardQueueEdit), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Disabled: !m.Board.QueueActions.Can(session.QueueActionEdit, item)}).Clicked() {
									m.OnQueueEdit(item.ID, item.Preview)
								}
							}
							if m.OnQueueSteer != nil && item.ID != "" {
								if coss.Button(c, i18n.T(i18n.KeyPlanBoardQueueSteer), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Disabled: !m.Board.QueueActions.Can(session.QueueActionSteer, item)}).Clicked() {
									m.OnQueueSteer(item.ID)
								}
							}
							if m.OnQueueRemove != nil && item.ID != "" {
								if coss.Button(c, i18n.T(i18n.KeyPlanBoardQueueRemove), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Disabled: !m.Board.QueueActions.Can(session.QueueActionRemove, item)}).Clicked() {
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
				ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
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
						ui.Row(c).Gap(design.Sp(2)).Padding(design.Sp(0.75), design.Sp(0.5), design.Sp(0.75), design.Sp(0.5)).FillWidth().AlignItems(ui.Center).Children(func() {
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
			ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					coss.Input(c, m.GoalDraft, coss.InputOptions{
						Size:        coss.InputSM,
						Placeholder: i18n.T(i18n.KeyPlanBoardGoalDraftPlaceholder),
					})
				})
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalCreate), buttonGhost, false).Clicked() {
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
	ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).Padding(design.Sp(2), design.Sp(2.5), design.Sp(2), design.Sp(2.5)).Radius(design.RadiusMD).
		Border(design.BorderWidth, subtleBorder).Background(cardBG).Children(func() {
		ui.Text(c, fmt.Sprintf("[%s] %s", status, g.Objective)).FontSize(theme.InterfaceSize(12.5)).TextColor(fg)
		if g.BlockedReason.Code != "" || g.BlockedReason.Message != "" {
			reason := g.BlockedReason.Message
			if g.BlockedReason.Code != "" {
				reason = "[" + g.BlockedReason.Code + "] " + reason
			}
			ui.Text(c, reason).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		if m.GoalDraft != nil && m.OnEditGoal != nil {
			ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					coss.Input(c, m.GoalDraft, coss.InputOptions{
						Size:        coss.InputSM,
						Placeholder: i18n.T(i18n.KeyPlanBoardGoalDraftPlaceholder),
					})
				})
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalEdit), buttonGhost, false).Clicked() {
					obj := strings.TrimSpace(*m.GoalDraft)
					if obj != "" {
						m.OnEditGoal(obj)
					}
				}
			})
		}
		ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			resume := g.Phase == "paused" || (g.Phase == "active" && g.Activation == "disarmed")
			if resume {
				if m.OnResumeGoal != nil {
					if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalResume), buttonGhost, false).Clicked() {
						m.OnResumeGoal()
					}
				}
			} else if g.Phase == "active" && g.Activation == "armed" && m.OnPauseGoal != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalPause), buttonGhost, false).Clicked() {
					m.OnPauseGoal()
				}
			}
			if m.OnCompleteGoal != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalComplete), buttonPrimary, false).Clicked() {
					m.OnCompleteGoal()
				}
			}
			if m.OnClearGoal != nil {
				if stdButton(c, palette, i18n.T(i18n.KeyPlanBoardGoalClear), buttonDanger, false).Clicked() {
					m.OnClearGoal()
				}
			}
		})
	})
}

func paintPlanSection(c *ui.Context, title string, muted, fg ui.Color, body func()) {
	_ = fg
	ui.Column(c).Gap(design.Sp(1.5)).FillWidth().MinWidth(0).Children(func() {
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
	ui.Column(c).FillWidth().Gap(design.Sp(1.5)).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyPlanBoardQueueEdit)).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		if e.Ready {
			draft := e.Text
			coss.Textarea(c, &draft, coss.TextareaOptions{
				Key:      "queue-edit-" + e.ItemID,
				Size:     coss.InputSM,
				ReadOnly: e.Saving,
				MaxLines: 6,
			}).Lines(2, 6).OnChange(func() {
				if m.OnChangeQueueEdit != nil {
					m.OnChangeQueueEdit(e.Token, draft)
				}
			})
		}
		if e.Error != "" {
			ui.Text(c, e.Error).FontSize(theme.InterfaceSize(11)).TextColor(fg)
		}
		ui.Row(c).Gap(design.Sp(2)).Children(func() {
			if e.Ready && m.OnSaveQueueEdit != nil && coss.Button(c, queueSaveEditCopy(), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonXS, Label: queueSaveEditCopy(), Disabled: e.Saving || strings.TrimSpace(e.Text) == ""}).Clicked() {
				m.OnSaveQueueEdit(e.Token)
			}
			if m.OnCancelQueueEdit != nil && stdButton(c, palette, queueCancelEditCopy(), buttonGhost, e.Saving).Label(queueCancelEditCopy()).Clicked() {
				m.OnCancelQueueEdit(e.Token)
			}
		})
	})
}
