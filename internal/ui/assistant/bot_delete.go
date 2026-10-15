package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
)

const botDeleteConfirmLabel = "永久删除此 Bot 及所列关联数据 / Permanently delete reviewed Bot"
const botDeleteDangerAcknowledgement = "我当次确认上述不可恢复删除和全部级联影响 / I confirm irreversible deletion now"
const botDeleteEffectsWarning = "危险操作：删除目标资料及不足 2 成员的群，移除群成员和 peer 来源，删除所属例行任务及 notepad 整个文件，取消相关未完成任务并清理关联 inbox；审计追加且最多保留 200 条。notepad 文件直接移除，无回收站或 Undo，其执行时全部内容可能不可恢复。会话历史和 bot/user memory 保留，不代表配置、任务状态或 notepad 全部可恢复。Host 会尝试停止受删目标并释放托管 presentation；解散群的成员会话仍隐藏保留。预览/提交前复核与删除不在同一原子锁：任务、inbox 及 notepad 仍可能变化，执行时按 Host 当前关联状态处理。可能先停止/取消或改 presentation，随后写入失败；仅尝试恢复 presentation，不承诺全部回滚。未知回包不等于无副作用，也不能安全重试。"

type BotDeleteModel struct {
	State                                    dshbotstore.BotDeleteSnapshot
	HostLabel, SourceID                      string
	OnRefresh, OnReview, OnConfirm, OnCancel func()
	OnIdentity                               func(id, name string)
	OnAcknowledge                            func(bool)
}

func botDeleteBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.BotDelete.OnRefresh == nil {
		return
	}
	botDeleteView(c, m.BotDelete, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func botDeleteReviewText(host string, p pluginrpc.DshbotBotDeletePlan) string {
	return fmt.Sprintf("Host: %s · catalog revision %d · kind: %s\n精确目标 ID: %s\n稳定名称 / Exact stable name: %s\n显示名 / Display title: %s\n原 canonical session: %s\n完整当前删除影响 / Complete current deletion effects:\n%s", host, p.Catalog.Revision, p.Target.Kind, p.Target.ID, p.Target.Name, p.Target.Title, p.Target.SessionID, string(p.Impact))
}
func botDeleteReviewConfirmed(r *dshbotstore.BotDeleteReview) bool {
	return r != nil && r.Acknowledged && r.TypedID == r.Plan.Target.ID && r.TypedName == r.Plan.Target.Name && r.TypedID != "" && r.TypedName != ""
}
func botDeleteView(c *ui.Context, m BotDeleteModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("native-bot-delete")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "删除选中 Bot 或群聊 / Delete selected Bot or room").FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "先读取完整影响；默认取消，读取预览不会删除。此入口不会删除其它选中目标，不额外发送 stop、prompt 或打开会话。").FontSize(11).TextColor(muted)
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(12).TextColor(muted).Selectable()
		}
		for _, w := range s.Warnings {
			ui.Text(c, "Host warning: "+w).TextColor(accent).Selectable()
		}
		if s.Completed {
			ui.Text(c, botDeleteReviewText(m.HostLabel, s.Plan)).FontSize(11).TextColor(fg).Selectable()
			return
		}
		if m.OnRefresh != nil && ui.Button(c, "只读刷新删除预览或结果 / Read deletion preview or outcome").Disabled(blocked || s.Scope == "" || m.SourceID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Uncertain {
			ui.Text(c, "原 attempt 保持未知且不可重发；目录存在或不存在均不证明文件、任务与运行副作用的结果。仅本进程保存锁定状态。不要把重启当作安全重试证明。").TextColor(accent).Selectable()
			if s.Pending != nil {
				ui.Text(c, botDeleteReviewText(m.HostLabel, s.Pending.Plan)).FontSize(11).TextColor(fg).Selectable()
			}
			if s.Observation != "" {
				ui.Text(c, s.Observation).FontSize(11).TextColor(fg).Selectable()
			}
			return
		}
		if !s.Loaded {
			ui.Text(c, "必须读取当前完整预览和写权限 / Full current preview and authority required").FontSize(11).TextColor(muted)
			return
		}
		if !s.Writable {
			ui.Text(c, "Host 只读或写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Review == nil {
			ui.Text(c, botDeleteReviewText(m.HostLabel, s.Plan)).FontSize(11).TextColor(fg).Selectable()
			if m.OnReview != nil && ui.Button(c, "审阅危险删除 / Review destructive deletion").Disabled(blocked || !s.Writable).Clicked() {
				m.OnReview()
			}
			return
		}
		r := s.Review
		effects, confirmLabel, cancelLabel, idLabel := botDeleteReviewedLabels(r.Plan.Target.Kind)
		ui.Column(c.Key("bot-delete-confirmation")).Border(1, accent).Radius(8).Padding(12).Gap(8).Children(func() {
			ui.Text(c, "删除前当次确认，默认取消 / Destructive confirmation; cancel by default").Bold().TextColor(accent)
			if m.OnCancel != nil && ui.Button(c, cancelLabel).Disabled(blocked).Clicked() {
				m.OnCancel()
			}
			ui.Text(c, botDeleteReviewText(m.HostLabel, r.Plan)).FontSize(12).TextColor(fg).Selectable()
			ui.Text(c, effects).FontSize(12).TextColor(accent).Selectable()
			ui.Text(c, "notepad 按列出的 Host Home 相对文件名删除当前全部内容，无内容 revision；若其它 routine 共用相同规范化文件名，也会失去该文件。此列表不是文件内容备份。").FontSize(11).TextColor(accent)
			name := r.TypedName
			ui.TextInput(c.Key("bot-delete-exact-name"), &name).Label("输入上方精确稳定名称 / Type exact stable name").FillWidth().Disabled(blocked).OnChange(func() {
				if m.OnIdentity != nil {
					m.OnIdentity(r.TypedID, name)
				}
			})
			id := r.TypedID
			ui.TextInput(c.Key("bot-delete-exact-id"), &id).Label(idLabel).FillWidth().Disabled(blocked).OnChange(func() {
				if m.OnIdentity != nil {
					m.OnIdentity(id, r.TypedName)
				}
			})
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botDeleteDangerAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if m.OnConfirm != nil && ui.Button(c, confirmLabel).Disabled(blocked || !s.Writable || !botDeleteReviewConfirmed(r)).Clicked() {
				m.OnConfirm()
			}
		})
	})
}
