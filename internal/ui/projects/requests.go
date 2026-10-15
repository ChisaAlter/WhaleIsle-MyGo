package projects

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
)

func requestsView(c *ui.Context, m Model) {
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("返回项目", "Back to Projects")).Clicked() && m.OnCloseRequests != nil {
			m.OnCloseRequests()
		}
		if ui.Button(c, tr("打开 Lead", "Open Lead")).Disabled(m.Busy).Clicked() && m.OnOpen != nil {
			m.OnOpen(m.RequestProjectID)
		}
	})
	ui.Text(c, tr("用户要求与交付", "User requests and deliveries")).FontSize(13)
	ui.Text(c, tr("此处显示用户目标及交付；Team任务板显示成员执行工作。", "These cards track user goals and delivery; the Team board tracks member work.")).FontSize(11)
	if m.RequestCard != nil {
		requestDetailView(c, m)
		return
	}
	ui.Row(c).Gap(3).Children(func() {
		for _, tab := range []struct{ kind, zh, en string }{{"current", "当前", "Current"}, {"completed", "已交付", "Delivered"}, {"archived", "已归档", "Archived"}} {
			tab := tab
			label := tr(tab.zh, tab.en)
			if tab.kind == m.RequestKind {
				label = "● " + label
			}
			if ui.Button(c, label).Disabled(m.Busy).Clicked() && m.OnRequestList != nil {
				m.OnRequestList(tab.kind)
			}
		}
	})
	ui.Text(c, fmt.Sprintf(tr("共%d项 · 第%d页", "%d requests · page %d"), m.RequestPage.Total, m.RequestPageNumber)).FontSize(11)
	if m.RequestLoaded && len(m.RequestPage.Items) == 0 {
		ui.Text(c, tr("此列表暂无用户要求。在Lead主对话中提出目标。", "No user requests in this list. Describe your goal in the Lead conversation.")).FontSize(12)
	}
	ui.Scroll(c).Height(280).Children(func() {
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			for _, card := range m.RequestPage.Items {
				card := card
				if ui.Button(c, card.Title).FillWidth().Disabled(m.Busy).Clicked() && m.OnRequestGet != nil {
					m.OnRequestGet(card.ID)
				}
				ui.Text(c, fmt.Sprintf(tr("第%d版 · %s", "Revision %d · %s"), card.Revision, requestStatus(card.Status))).FontSize(11)
				ui.Text(c, card.GoalPreview).FontSize(12)
				if card.Delivery != nil {
					ui.Text(c, card.Delivery.SummaryPreview).FontSize(11)
					if !card.Delivery.Applicable {
						ui.Text(c, tr("交付依据已变化，需复核。", "Delivery basis changed; review is required.")).FontSize(11)
					}
				}
				requestArchiveButton(c, m, card)
			}
		})
	})
	ui.Row(c).Gap(4).Children(func() {
		if ui.Button(c, tr("上一页", "Previous")).Disabled(m.Busy || m.RequestPageNumber <= 1).Clicked() && m.OnRequestPage != nil {
			m.OnRequestPage(-1)
		}
		if ui.Button(c, tr("下一页", "Next")).Disabled(m.Busy || m.RequestPage.NextCursor == nil).Clicked() && m.OnRequestPage != nil {
			m.OnRequestPage(1)
		}
	})
}
func requestArchiveButton(c *ui.Context, m Model, card projectflow.RequestCard) {
	label := tr("归档已交付要求", "Archive delivered request")
	if card.ArchivedAt != nil {
		label = tr("恢复到任务列表", "Restore to task list")
	}
	allowed := m.Listing.Available && !m.Busy && m.RequestMutable && (card.ArchivedAt != nil || card.Status == "completed")
	if ui.Button(c, label).Disabled(!allowed).Clicked() && m.OnRequestArchive != nil {
		m.OnRequestArchive(card.ID, card.ArchivedAt == nil)
	}
}
func requestDetailView(c *ui.Context, m Model) {
	card := m.RequestCard
	if m.AssetNote != "" {
		ui.Text(c, m.AssetNote).FontSize(11)
	}
	if m.AssetDiff != nil {
		assetDiffView(c, m)
		return
	}
	if ui.Button(c, tr("打开项目目录", "Open Project directory")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnProjectDirectory != nil {
		m.OnProjectDirectory()
	}
	if ui.Button(c, tr("返回要求列表", "Back to request list")).Disabled(m.Busy).Clicked() && m.OnBackRequests != nil {
		m.OnBackRequests()
	}
	ui.Text(c, card.Title).FontSize(13)
	ui.Text(c, fmt.Sprintf(tr("第%d版 · %s", "Revision %d · %s"), card.Revision, requestStatus(card.Status))).FontSize(12)
	requestArchiveButton(c, m, *card)
	ui.Scroll(c).Height(280).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Children(func() {
			ui.Text(c, card.Goal).FontSize(12)
			if card.AuthorityError != "" {
				ui.Text(c, tr("授权记录问题：", "Authorization issue: ")+card.AuthorityError).FontSize(12)
			}
			if card.HandoverError != "" {
				ui.Text(c, tr("交付未完成：", "Delivery incomplete: ")+card.HandoverError).FontSize(12)
				ui.Text(c, tr("恢复指令只读取已保存成果，不重新执行后台工作。草稿仍需你发送。", "The recovery request reads saved results without rerunning background work. You still send the draft.")).FontSize(11)
				if ui.Button(c, tr("准备交付恢复草稿", "Prepare handover recovery draft")).Disabled(m.Busy || !m.Listing.Available || !m.RequestMutable || card.AuthorityError != "").Clicked() && m.OnHandoverRetryDraft != nil {
					m.OnHandoverRetryDraft()
				}
			}
			for _, condition := range card.Conditions {
				ui.Text(c, tr("验收条件：", "Acceptance condition: ")+condition).FontSize(11)
			}
			for _, constraint := range card.Constraints {
				ui.Text(c, tr("约束：", "Constraint: ")+constraint).FontSize(11)
			}
			if card.Delivery == nil {
				ui.Text(c, tr("尚无最终交付，成员报告可能仍已保存。", "No final delivery yet; member reports may already be saved.")).FontSize(12)
			} else {
				delivery := card.Delivery
				for _, file := range delivery.Artifacts {
					file := file
					label := file.Title
					if label == "" {
						label = file.Path
					}
					if ui.Button(c, tr("打开文件：", "Open file: ")+label).FillWidth().Disabled(m.Busy || !m.Listing.Available || m.OnResolvedFile == nil).Clicked() && m.OnArtifact != nil {
						m.OnArtifact(file, m.OnResolvedFile)
					}
					ui.Text(c, file.DelegationRef).FontSize(11)
				}
				ui.Text(c, tr("交付：", "Delivery: ")+delivery.Outcome).FontSize(12)
				if !delivery.Applicable {
					ui.Text(c, tr("此交付不再适用于当前要求或依据。", "This delivery no longer applies to the current requirement or evidence.")).FontSize(12)
				}
				ui.Text(c, delivery.Summary).FontSize(12)
				for _, e := range delivery.Evidence {
					ui.Text(c, tr("验证：", "Evidence: ")+e).FontSize(11)
				}
				for _, issue := range delivery.RemainingIssues {
					ui.Text(c, tr("剩余问题：", "Remaining issue: ")+issue).FontSize(11)
				}
			}
			ui.Text(c, tr("关联工作", "Associated work")).FontSize(12)
			for _, work := range card.Work {
				work := work
				if ui.Button(c, tr("查看当前工作区差异", "View current workspace diff")).Disabled(m.Busy || !m.Listing.Available).Clicked() && m.OnWorkDiff != nil {
					m.OnWorkDiff(work.WorkstreamID)
				}
				ui.Text(c, work.Title+" · "+stateLabel(work.Status)).FontSize(11)
				ui.Text(c, work.MemberName+" · "+work.WorkstreamID+" · "+work.DelegationRef).FontSize(11)
				if work.BlockedReason != "" {
					ui.Text(c, work.BlockedReason).FontSize(11)
				}
				if work.Summary != "" {
					ui.Text(c, work.Summary).FontSize(11)
				}
			}
			ui.Text(c, tr("要求版本", "Requirement versions")).FontSize(12)
			for _, version := range card.Versions {
				ui.Text(c, fmt.Sprintf("%d · %s", version.Revision, version.Title)).FontSize(11)
				ui.Text(c, version.Goal).FontSize(11)
			}
		})
	})
}

func requestStatus(status string) string {
	if status == "completed" {
		return tr("已交付", "Delivered")
	}
	if status == "pendingDelivery" {
		return tr("等待交付", "Awaiting delivery")
	}
	return stateLabel(status)
}
