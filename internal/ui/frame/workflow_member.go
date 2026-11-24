package frame

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

func workflowMemberDialog(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	v := m.Store.WorkflowMemberSnapshot()
	if !v.Open {
		return
	}
	t := tk(c)
	open := true
	_, height := viewportDIP(c)
	panelHeight := min(float32(700), max(float32(280), height-design.Sp(16)))
	bodyHeight := max(float32(160), panelHeight-design.Sp(28))
	coss.Dialog(c, &open, coss.DialogOptions{
		Title:         v.Label + " · Workflow",
		Description:   v.ParentID + " → " + v.ChildID + " · one-shot",
		MaxWidth:      940,
		NoClose:       true,
		FooterVariant: coss.FooterBare,
		Footer: func() {
			if coss.Button(c, archiveText("关闭", "Close"), coss.ButtonOptions{Variant: coss.ButtonOutline}).Clicked() {
				open = false
			}
		},
	}, func() {
		ui.Column(c).FillWidth().Height(bodyHeight).Gap(design.Sp(2)).Children(func() {
			ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				status := archiveText("已暂停或结束跟进", "Observation paused or ended")
				if v.Busy {
					status = archiveText("正在读取…", "Loading…")
				} else if v.Process.Live {
					status = archiveText("实时过程", "Live process")
				}
				frameText(c, status, design.TextXS, design.WeightNormal, t.MutedForeground).Grow(1)
				if coss.Button(c, archiveText("较早记录", "Earlier records"), coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonXS,
					Disabled: v.Busy || !v.Process.HasMore,
				}).Clicked() {
					m.Store.EarlierWorkflowMember(v.Token)
				}
				if coss.Button(c, archiveText("回到最新 / 重连", "Latest / reconnect"), coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonXS,
					Disabled: v.Busy || !m.Store.WorkflowMemberCanReconnect(v.Token),
				}).Clicked() {
					// reconnect uses PlanHistoryBinding()
					m.Store.OpenWorkflowMember(m.Store.PlanHistoryBinding(), v.ParentID, v.RunID, v.MemberSeq, v.ChildID, v.Label)
				}
				if coss.Button(c, archiveText("复制持久记录", "Copy durable records"), coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonXS,
					Disabled: len(v.Process.Records) == 0,
				}).Clicked() {
					mygo.Clipboard.WriteText(session.WorkflowProcessText(v))
				}
			})
			if v.Process.Error != "" {
				coss.Alert(c, "", v.Process.Error, coss.AlertOptions{Variant: coss.AlertError})
			}
			if v.Process.Truncated {
				frameText(c, archiveText("仅保留最近400条；较早记录可分页读取。", "Showing the latest 400 records; earlier history remains pageable."), design.TextXS, design.WeightNormal, t.MutedForeground)
			}
			rawKey := fmt.Sprintf("workflow-member-raw:%s:%s:%d", v.ParentID, v.ChildID, v.Token)
			rawOpen := false
			if m.ThinkingExpanded != nil {
				if *m.ThinkingExpanded == nil {
					*m.ThinkingExpanded = map[string]bool{}
				}
				rawOpen = (*m.ThinkingExpanded)[rawKey]
				if coss.Button(c, archiveText("显示/隐藏原始持久记录", "Show/hide durable raw records"), coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonXS,
				}).Clicked() {
					(*m.ThinkingExpanded)[rawKey] = !rawOpen
				}
			}
			coss.ScrollArea(c.Key(fmt.Sprintf("workflow-member:%s:%s:%d", v.ParentID, v.ChildID, v.Token)), coss.ScrollAreaOptions{}, func() {
				body := session.WorkflowMemberDisplayText(v)
				if body == "" {
					body = archiveText("尚无可显示内容。", "No visible content yet.")
				}
				ui.Column(c).FillWidth().Gap(design.Sp(3)).Children(func() {
					frameText(c, body, design.TextXS, design.WeightNormal, t.Foreground).Selectable()
					if rawOpen && v.PresentationReady {
						frameText(c, session.WorkflowProcessText(v), design.TextXS, design.WeightNormal, t.MutedForeground).Selectable()
					}
				})
			}).FillWidth().Grow(1).MinHeight(0)
		})
	})
	if !open {
		m.Store.CloseWorkflowMember(v.Token)
	}
}
