package chat

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"math"
	"strings"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/workflowrun"
)

func workflowText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

func workflowStatus(s string) string {
	switch s {
	case "running":
		return workflowText("运行中", "Running")
	case "completed":
		return workflowText("已完成", "Completed")
	case "failed":
		return workflowText("失败", "Failed")
	case "cancelled":
		return workflowText("已取消", "Cancelled")
	case "interrupted":
		return workflowText("已中断", "Interrupted")
	}
	return s
}
func workflowAbnormal(s string) bool { return s == "failed" || s == "cancelled" || s == "interrupted" }
func WorkflowRunView(c *ui.Context, run workflowrun.Run, m Model, st ToolCardStyle) {
	// State belongs to the exact Host/session binding, run and phase, not labels.
	key := fmt.Sprintf("workflow:%q:%q", m.Snapshot.PlanHistoryBinding, run.ID)
	card := ui.Column(c.Key(key)).FillWidth().Gap(6).Padding(10).Border(1, st.Border).Radius(8)
	disclosures := ui.Local(card, "status-disclosures", func() workflowDisclosures { return newWorkflowDisclosures(run) })
	phaseStartedCycle := disclosures.syncPhases(run)
	facts := workflowRunFacts(run)
	name := run.Name
	if name == "" {
		name = "Workflow"
	}
	count := 0
	for _, p := range run.Phases {
		count += len(p.Members)
	}
	card.Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			header := ui.Button(c, "").Label("Workflow " + name).Grow(1).FontSize(13)
			clicked := header.Clicked()
			// Keep a pending header press through release. Keyboard departure
			// still settles the deferred clean collapse of its content.
			disclosures.run.advance(facts, disclosures.run.content.FocusWithin(c) || disclosures.run.pendingCleanCollapse && (header.Pressed() || clicked))
			if phaseStartedCycle && facts.mode != "clean" {
				disclosures.run.open, disclosures.run.pendingCleanCollapse = true, false
			}
			if clicked {
				disclosures.run.toggle()
			}
			header.Children(func() { ui.Text(c, workflowDisclosureArrow(disclosures.run.open)+name) })
			ink := st.FG
			if workflowAbnormal(run.Status) {
				ink = st.Err
			}
			ui.Text(c, workflowStatus(run.Status)).FontSize(11).TextColor(ink)
			ui.Text(c, fmt.Sprintf("%d" + " " + i18n.T(i18n.KeyChatWorkflowMembers), count)).FontSize(11).TextColor(st.Muted)
		})
		if !disclosures.run.open {
			for _, phase := range run.Phases {
				disclosures.phases[phase.Key].advance(workflowPhaseFacts(phase), false)
			}
			return
		}
		ui.Column(c.Key("run-content")).Bind(&disclosures.run.content).FillWidth().Gap(6).Children(func() {
			if count == 0 {
				ui.Text(c, i18n.T(i18n.KeyChatWorkflowNoMember)).FontSize(11).TextColor(st.Muted)
			}
			for _, phase := range run.Phases {
				p := phase
				phaseName := i18n.T(i18n.KeyChatWorkflowUngrouped)
				if p.Name != nil {
					phaseName = *p.Name
					if phaseName == "" {
						phaseName = i18n.T(i18n.KeyChatWorkflowEmptyPhase)
					}
				}
				pk := key + ":" + p.Key
				phaseState := disclosures.phases[p.Key]
				ui.Column(c.Key(pk)).FillWidth().Gap(5).Padding(5, 0, 5, 12).Children(func() {
					header := ui.Button(c, "").Label("Workflow phase " + phaseName).FontSize(11)
					clicked := header.Clicked()
					phaseState.advance(workflowPhaseFacts(p), phaseState.content.FocusWithin(c) || phaseState.pendingCleanCollapse && (header.Pressed() || clicked))
					if clicked {
						phaseState.toggle()
					}
					header.Children(func() {
						ui.Text(c, workflowDisclosureArrow(phaseState.open)+phaseName+" · "+workflowPhaseSummary(p.Members))
					})
					if !phaseState.open {
						return
					}
					ui.Column(c.Key("members")).Bind(&phaseState.content).FillWidth().Gap(2).Padding(0, 0, 0, 16).Children(func() {
						for _, member := range p.Members {
							ch := member
							label := ch.Label
							if label == "" {
								label = i18n.T(i18n.KeyChatWorkflowUnnamed)
							}
							available := m.FeedbackStore != nil && session.WorkflowMemberAvailable(m.Snapshot.Items, m.Snapshot.SelectedID, ch)
							paintWorkflowMember(c, ch, label, available, st, func() {
								m.FeedbackStore.OpenWorkflowMember(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, run.ID, ch.Seq, ch.ChildID, label)
							})
						}
					})
				})
			}
		})
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, i18n.T(i18n.KeyChatWorkflowCopy)).FontSize(10).Clicked() {
				raw, _ := json.MarshalIndent(run.Events, "", "  ")
				mygo.Clipboard.WriteText(string(raw))
			}
			ui.Text(c, i18n.T(i18n.KeyChatWorkflowEndedNote)).FontSize(10).TextColor(st.Muted)
		})
	})
}

// Completed members are quiet while work or failures remain. An interruption
// explicitly includes already completed work; an entirely clean phase shows
// its completed count, including zero, as WorkflowRunPanel.phaseStatusSummary.
func workflowPhaseSummary(members []workflowrun.Member) string {
	counts := map[string]int{}
	for _, member := range members {
		counts[member.Status]++
	}
	statuses := []string{}
	for _, status := range []string{"running", "failed", "cancelled", "interrupted"} {
		if counts[status] > 0 {
			statuses = append(statuses, status)
		}
	}
	if len(statuses) == 0 {
		statuses = []string{"completed"}
	} else if counts["interrupted"] > 0 && counts["completed"] > 0 {
		statuses = append([]string{"completed"}, statuses...)
	}
	parts := make([]string, 0, len(statuses))
	for _, status := range statuses {
		parts = append(parts, fmt.Sprintf("%s %d", workflowStatus(status), counts[status]))
	}
	return strings.Join(parts, " · ")
}

func paintWorkflowMember(c *ui.Context, member workflowrun.Member, label string, available bool, st ToolCardStyle, open func()) {
	owner := ui.Row(c.Key(fmt.Sprintf("member:%.0f", member.Seq))).FillWidth().MinWidth(0)
	action := ui.Local(owner, "action", func() ui.Handle { return ui.Handle{} })
	owner.Children(func() {
		// A row that stops being navigable retains the same focused button until
		// blur. Its action is guarded by current availability, not old focus.
		var row ui.Element
		if available || action.Focused(c) {
			row = ui.Button(c.Key("member-row"), "").Label(label)
			if available {
				row.Label(workflowText("打开 ", "Open ") + label)
			}
		} else {
			row = ui.Row(c.Key("member-row")).Label(label)
		}
		size := st.FontSize
		if size <= 0 {
			size = 14
		}
		row.Bind(action).FillWidth().MinWidth(0).MinHeight(size + 10).Padding(0).Gap(12).
			AlignItems(ui.Center).Radius(4).BorderWidth(0).Background(ui.Transparent).FocusRing(false)
		row.Children(func() {
			ui.Box(c).Width(16).Height(size + 10).Shrink(0).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				paintWorkflowStateDot(c, member.Status)
			})
			labelWrap := ui.Row(c).Grow(1).MinWidth(0).Height(size+10).Padding(0, 2).Radius(4).AlignItems(ui.Center)
			if row.FocusVisible() {
				labelWrap.Border(1, c.Theme().Focus)
			}
			labelWrap.Children(func() {
				text := ui.Text(c, label).Grow(1).MinWidth(0).MaxLines(1).FontSize(size).FixedLineHeight(size + 10).TextColor(st.Muted)
				if available {
					text.TextColor(st.Accent).FontWeight(500)
					if row.Hovered() || row.FocusVisible() {
						text.Underline()
					}
				}
			})
			ui.Text(c, workflowStatus(member.Status)).Width(64).Shrink(0).MaxLines(1).
				FontSize(size - 1).FixedLineHeight(size + 7).TextAlign(ui.End).TextColor(st.Muted)
		})
		if available && row.Clicked() {
			open()
		}
	})
}

// StateDot's ongoing ring is the only animated state. All solid states occupy
// a ten-DIP slot with a six-DIP core; elapsed time keeps new rings in phase.
func paintWorkflowStateDot(c *ui.Context, status string) {
	if status != "running" {
		ink := c.Theme().Success
		if status == "failed" {
			ink = c.Theme().Danger
		} else if status == "cancelled" || status == "interrupted" {
			ink = c.Theme().Warning
		}
		ui.Box(c).Width(10).Height(10).PassThrough().Children(func() {
			ui.Box(c).Width(6).Height(6).Margin(2).Radius(3).Background(ink)
		})
		return
	}
	reduced := c.Preferences().ReduceMotion
	ink := c.Theme().TextMuted
	ui.Box(c).Width(14).Height(14).PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, radius := r.X+r.W/2, r.Y+r.H/2, r.W*9.5/24
		var track ui.Path
		track.Circle(cx, cy, radius)
		p.StrokePath(&track, r.W*2/24, ink.Alpha(.25))
		angle, length := float64(0), float64(18)
		if !reduced {
			phase := float64(p.Now().UnixMilli()%1500) / 1500
			breath := (1 - math.Cos(phase*2*math.Pi)) / 2
			angle, length = phase*2*math.Pi-6*breath/9.5, 12+12*breath
			p.After(16 * time.Millisecond)
		} else {
			angle = -3 / 9.5
		}
		var arc ui.Path
		for step := 0; step <= 24; step++ {
			a := angle + float64(step)*length/(24*9.5)
			x, y := cx+radius*float32(math.Cos(a)), cy+radius*float32(math.Sin(a))
			if step == 0 {
				arc.MoveTo(x, y)
			} else {
				arc.LineTo(x, y)
			}
		}
		p.StrokePath(&arc, r.W*2/24, ink)
	})
}
