package chat

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/workflowrun"
)

func workflowStatus(s string) string {
	switch s {
	case "running":
		return "运行中 / Running"
	case "completed":
		return "已完成 / Completed"
	case "failed":
		return "失败 / Failed"
	case "cancelled":
		return "已取消 / Cancelled"
	case "interrupted":
		return "中断 / Interrupted"
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
			ui.Text(c, fmt.Sprintf("%d 成员 / members", count)).FontSize(11).TextColor(st.Muted)
		})
		if !disclosures.run.open {
			for _, phase := range run.Phases {
				disclosures.phases[phase.Key].advance(workflowPhaseFacts(phase), false)
			}
			return
		}
		ui.Column(c.Key("run-content")).Bind(&disclosures.run.content).FillWidth().Gap(6).Children(func() {
			if count == 0 {
				ui.Text(c, "尚无已启动成员 / No member has started").FontSize(11).TextColor(st.Muted)
			}
			for _, phase := range run.Phases {
				p := phase
				phaseName := "未分组 / Unassigned"
				if p.Name != nil {
					phaseName = *p.Name
					if phaseName == "" {
						phaseName = "空阶段名 / Empty phase"
					}
				}
				pk := key + ":" + p.Key
				phaseState := disclosures.phases[p.Key]
				summary := map[string]int{}
				for _, member := range p.Members {
					summary[member.Status]++
				}
				counts := []string{}
				for _, status := range []string{"running", "completed", "failed", "cancelled", "interrupted"} {
					if n := summary[status]; n > 0 {
						counts = append(counts, fmt.Sprintf("%s %d", workflowStatus(status), n))
					}
				}
				ui.Column(c.Key(pk)).FillWidth().Gap(5).Padding(5, 0, 5, 12).Children(func() {
					header := ui.Button(c, "").Label("Workflow phase " + phaseName).FontSize(11)
					clicked := header.Clicked()
					phaseState.advance(workflowPhaseFacts(p), phaseState.content.FocusWithin(c) || phaseState.pendingCleanCollapse && (header.Pressed() || clicked))
					if clicked {
						phaseState.toggle()
					}
					header.Children(func() {
						ui.Text(c, workflowDisclosureArrow(phaseState.open)+phaseName+" · "+strings.Join(counts, " · "))
					})
					if !phaseState.open {
						return
					}
					ui.Column(c.Key("members")).Bind(&phaseState.content).FillWidth().Gap(5).Children(func() {
						for _, member := range p.Members {
							ch := member
							label := ch.Label
							if label == "" {
								label = "未命名成员 / Unnamed member"
							}
							available := m.FeedbackStore != nil && session.WorkflowMemberAvailable(m.Snapshot.Items, m.Snapshot.SelectedID, ch)
							ui.Row(c.Key(fmt.Sprintf("member:%.0f", ch.Seq))).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Text(c, label).FontSize(12).TextColor(st.FG).Grow(1)
								ui.Text(c, workflowStatus(ch.Status)).FontSize(11).TextColor(st.Muted)
								action := ui.Button(c, "查看过程 / View process").FontSize(11)
								// A live member can finish while its action holds focus.
								// Retain that identity until blur, but never navigate it.
								action.Disabled(!available && !action.Focused())
								if action.Clicked() && available {
									m.FeedbackStore.OpenWorkflowMember(m.Snapshot.PlanHistoryBinding, m.Snapshot.SelectedID, run.ID, ch.Seq, ch.ChildID, label)
								}
							})
						}
					})
				})
			}
		})
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "复制运行记录 / Copy run records").FontSize(10).Clicked() {
				raw, _ := json.MarshalIndent(run.Events, "", "  ")
				mygo.Clipboard.WriteText(string(raw))
			}
			ui.Text(c, "已结束或不可用成员保留轨迹，不会重新启动。 / Unavailable members retain their recorded status.").FontSize(10).TextColor(st.Muted)
		})
	})
}
