package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strconv"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

const botCreateAcknowledgement = "我理解 Bot 创建与会话影响 / I understand Bot and session effects"
const botCreateConfirmLabel = "确认创建并打开 Bot / Confirm create and open Bot"

type BotCreateModel struct {
	State                                                                 dshbotstore.BotCreateSnapshot
	HostLabel                                                             string
	OnRefresh, OnStart, OnReview, OnConfirm, OnDismiss, OnDiscard, OnOpen func()
	OnChange                                                              func(field, value string)
	OnAcknowledge                                                         func(bool)
	OnAdopt                                                               func(id string)
}

func botCreateBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.BotCreate.OnRefresh == nil {
		return
	}
	botCreateView(c, m.BotCreate, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func botCreateReviewText(host string, cmd pluginrpc.DshbotBotCreateCommand) string {
	d := cmd.Draft
	return fmt.Sprintf("Host: %s · revision %d\nBot name: %s\nDisplay title: %s\nDescription: %s\nWorkspace ID: %s\nScratch cwd: %s\nBot ID: %s\nCanonical session: %s\nModel provider: %s\nModel ID: %s\nReasoning effort: %s\nmemoryReview: %t · notify: %t\nAvatar: blob circle/blue\nTools / Skills / MCP: 全部 / all (empty name filters)\nPeer message/task sources: []\nobjective: empty · objectiveMaxRounds: 16\nSection: none · pinned: false · hidden: false\nPersonal Bot defaults: memberBotIds [] · maxRounds 3 · maxSpeaks 10", host, cmd.Expected.Revision, d.Name, d.Title, d.Description, d.WorkspaceID, d.ScratchCwd, cmd.ID, cmd.SessionID, d.Provider, d.Model, d.ReasoningEffort, d.MemoryReview, d.Notify)
}
func botCreateView(c *ui.Context, m BotCreateModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("native-bot-create")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, "新建个人 Bot / Create personal Bot").FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "创建 Bot 资料与专属会话。请填写真实模型 provider / model ID；Host 会校验模型配置。更多头像、能力和来源配置仍可使用 Bot Web。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, "刷新 Bot 创建目录 / Refresh Bot creation catalog").Disabled(blocked || s.Scope == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "先读取完整目录、revision 与写权限 / Read current catalog and authority first").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else if !s.Writable {
			ui.Text(c, "Host 写权限未核实或目录只读 / Write authority unavailable").TextColor(accent)
		}
		if s.Created.ID != "" {
			ui.Text(c, fmt.Sprintf("已创建 / Created: %s (%s)\nCanonical session: %s", s.Created.Name, s.Created.ID, s.Created.SessionID)).TextColor(fg).Selectable()
			ui.Text(c, "打开仅核实或恢复这个 Bot 的专属会话。打开失败会保留创建结果，不重新创建。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if m.OnOpen != nil && ui.Button(c, "打开已创建 Bot / Open created Bot").Disabled(blocked).Clicked() {
				m.OnOpen()
			}
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, "新建个人 Bot / New personal Bot").Disabled(blocked || !s.Loaded || !s.Writable || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		ui.Text(c, fmt.Sprintf("Host: %s · revision %d", m.HostLabel, s.Catalog.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		for _, f := range []struct{ id, label, value string }{{"name", "稳定名称 / Stable Bot name (≤120)", d.Name}, {"title", "显示标题 / Display title (≤120)", d.Title}, {"workspaceId", "工作区 ID / Workspace ID (空=临时目录)", d.WorkspaceID}, {"scratchCwd", "临时会话目录 / Scratch cwd (无工作区时必填)", d.ScratchCwd}, {"provider", "模型提供方 ID / Provider ID (必填，≤200)", d.Provider}, {"model", "模型 ID / Model ID (必填，≤500)", d.Model}, {"reasoningEffort", "推理强度 ID / Reasoning effort (可空，≤100)", d.ReasoningEffort}} {
			f := f
			value := f.value
			ui.TextInput(c.Key("new-bot-"+f.id), &value).Label(f.label).FillWidth().Disabled(disabled).OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, value)
				}
			})
		}
		description := d.Description
		ui.TextArea(c.Key("new-bot-description"), &description).Label("Bot 说明 / Description (≤12000)").Lines(3, 8).FillWidth().Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("description", description)
			}
		})
		for _, f := range []struct {
			id, label string
			value     bool
		}{{"memoryReview", "允许后续记忆复盘 / Memory review", d.MemoryReview}, {"notify", "允许后续完成通知 / Notifications", d.Notify}} {
			f := f
			value := f.value
			if ui.Checkbox(c.Key("new-bot-"+f.id), &value, f.label).Disabled(disabled).Changed() && m.OnChange != nil {
				m.OnChange(f.id, strconv.FormatBool(value))
			}
		}
		if s.Uncertain {
			ui.Text(c, "创建结果未知：保留并锁定草稿与身份，只读核对，不自动重试。没有记录也不能重建。关闭或隐藏界面不取消已发送创建；保护仅保留在当前应用进程。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if row := s.Candidate; row != nil {
				ui.Text(c, fmt.Sprintf("精确匹配目录记录 / Exact matching record: %s (%s)\nCanonical session: %s", row.Name, row.ID, row.SessionID)).TextColor(fg).Selectable()
				if s.Pending != nil {
					ui.Text(c, botCreateReviewText(m.HostLabel, *s.Pending)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
				}
				ui.Text(c, "所有提交字段与该身份记录匹配；目录存在不证明会话创建完成。采用仅保留已有身份，之后单独打开会核实或恢复 canonical 会话。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if m.OnAdopt != nil && ui.Button(c, "采用核对后的 Bot / Adopt inspected Bot").Disabled(blocked).Clicked() {
					m.OnAdopt(row.ID)
				}
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, "丢弃 Bot 草稿 / Discard Bot draft").Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, "审阅 Bot 创建 / Review Bot creation").Disabled(disabled || !s.Loaded || !s.Writable).Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			ui.Column(c.Key("new-bot-confirmation")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, "创建前完整审阅 / Review new Bot").Bold().TextColor(fg)
				ui.Text(c, botCreateReviewText(m.HostLabel, r.Command)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "Host 将校验模型配置、写入 Bot 目录与审计记录、创建并打开专属会话，可能触发会话生命周期处理。非空 workspace 优先于 scratch cwd。后续 Bot 模型、工具、记忆复盘和通知可能产生费用或外部消息。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, "本次不发送 prompt、不调用模型生成、不启动 objective 或例行任务、不发送 IM。使用上方固定头像与全部能力；没有 peer 来源和目标。创建未知不自动重试；打开失败与创建失败分开处理。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, botCreateAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if m.OnDismiss != nil && ui.Button(c, "返回修改 Bot / Back to Bot draft").Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, botCreateConfirmLabel).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}
