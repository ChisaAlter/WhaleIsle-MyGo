package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

const botRouteAcknowledgement = "我理解模型、目录、新会话及目标可能执行的影响 / I understand model, directory, new-session and execution effects"

type BotRouteModel struct {
	State                                                         dshbotstore.RouteSnapshot
	HostLabel                                                     string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange                                                      func(pluginrpc.DshbotRouteValues)
	OnAcknowledge                                                 func(bool)
}

func botRouteModelText(v pluginrpc.DshbotRouteModel) string {
	if v.Provider == "" {
		return "沿用 Host 默认模型 / Host default"
	}
	return fmt.Sprintf("provider=%s · model=%s · effort=%s", v.Provider, v.Model, v.ReasoningEffort)
}
func botRouteWorkspaceText(id string, o pluginrpc.DshbotRouteOptions) string {
	if id == "" {
		return "无注册工作区 / Scratch: " + o.ScratchCwd
	}
	for _, w := range o.Workspaces {
		if w.ID == id {
			return fmt.Sprintf("%s · id=%s · path=%s", w.Title, w.ID, w.Path)
		}
	}
	return "id=" + id + " · Host 路径未核实 / Path unavailable"
}
func botRouteBlock(c *ui.Context, m BotRouteModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && !s.Uncertain && s.Review == nil
	ui.Column(c.Key("bot-route-editor")).Gap(7).Children(func() {
		ui.Text(c, "模型与工作区 / Model and workspace").FontSize(13).Bold().TextColor(fg)
		ui.Text(c, "选择当前 Host 的模型、推理强度和真实目录。完整资料无法无损保留时，请使用现有 Bot Web 编辑入口。").FontSize(11).TextColor(muted)
		if ui.Button(c, "刷新模型与工作区 / Refresh model and workspace").Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(11).TextColor(muted).Selectable()
		}
		for _, w := range s.Warnings {
			ui.Text(c, "Host warning: "+w).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, "先读回核实；未知结果保留并锁定草稿，不自动重试 / Read back before saving").TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		before, err := pluginrpc.DshbotRouteValue(s.Status)
		if err != nil {
			ui.Text(c, "字段不完整，请使用 Bot Web / Incomplete profile; use Bot Web").TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s · revision %d\n当前模型 / Current model: %s\n当前目录 / Current workspace: %s", m.HostLabel, s.Status.Bot.Name, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botRouteModelText(before.Model), botRouteWorkspaceText(before.WorkspaceID, s.Status.Options))).FontSize(12).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, "Host 写权限未核实 / Write authority unavailable").TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := s.Draft
		opts := s.Status.Options
		ui.Text(c, "模型草稿 / Model draft: "+botRouteModelText(draft.Model)).FontSize(12).TextColor(fg).Selectable()
		if !opts.ModelsKnown {
			ui.Text(c, "模型目录不可用；保留原配置，不猜测默认模型 / Model discovery unavailable").TextColor(accent)
		}
		modelLabels := []string{"沿用 Host 默认模型 / Host default"}
		modelChoices := map[string]pluginrpc.DshbotRouteModel{modelLabels[0]: {}}
		modelSelected := "当前保留配置 / Preserve current: " + botRouteModelText(draft.Model)
		if draft.Model.Provider == "" {
			modelSelected = modelLabels[0]
		}
		var effortOptions []string
		for i, o := range opts.Models {
			label := fmt.Sprintf("%d · %s · provider=%s · model=%s", i+1, o.Label, o.Provider, o.Model)
			modelLabels = append(modelLabels, label)
			modelChoices[label] = pluginrpc.DshbotRouteModel{Provider: o.Provider, Model: o.Model, ReasoningEffort: o.DefaultEffort}
			if o.Provider == draft.Model.Provider && o.Model == draft.Model.Model {
				modelSelected = label
				effortOptions = o.Efforts
			}
		}
		// Labels include a unique index; opaque provider/model IDs are never joined
		// into an ambiguous route key or split on slash/colon.
		if _, ok := modelChoices[modelSelected]; !ok {
			modelLabels = append(modelLabels, modelSelected)
		}
		ui.Select(c.Key("bot-route-model"), &modelSelected, modelLabels).FillWidth().Disabled(!editable || !opts.ModelsKnown).OnChange(func() {
			if v, ok := modelChoices[modelSelected]; ok && m.OnChange != nil {
				draft.Model = v
				m.OnChange(draft)
			}
		})
		ui.Text(c, "Host 当前默认值 / Current Host default: "+botRouteModelText(opts.Default)).FontSize(11).TextColor(muted).Selectable()
		effortLabels := []string{"模型默认推理强度 / Model default effort"}
		effortChoices := map[string]string{effortLabels[0]: ""}
		effortSelected := effortLabels[0]
		for i, e := range effortOptions {
			label := fmt.Sprintf("%d · %s", i+1, e)
			effortLabels = append(effortLabels, label)
			effortChoices[label] = e
			if draft.Model.ReasoningEffort == e {
				effortSelected = label
			}
		}
		if draft.Model.ReasoningEffort != "" && effortSelected == effortLabels[0] {
			effortSelected = "保留原强度 / Preserve: " + draft.Model.ReasoningEffort
			effortLabels = append(effortLabels, effortSelected)
		}
		ui.Select(c.Key("bot-route-effort"), &effortSelected, effortLabels).FillWidth().Disabled(!editable || !opts.ModelsKnown || draft.Model.Provider == "").OnChange(func() {
			if v, ok := effortChoices[effortSelected]; ok && m.OnChange != nil {
				draft.Model.ReasoningEffort = v
				m.OnChange(draft)
			}
		})
		ui.Text(c, "目录草稿 / Workspace draft: "+botRouteWorkspaceText(draft.WorkspaceID, opts)).FontSize(12).TextColor(fg).Selectable()
		workspaceLabels := []string{}
		workspaceChoices := map[string]string{}
		workspaceSelected := "保留现有目录 / Preserve current: " + draft.WorkspaceID
		if opts.ScratchCwd != "" {
			label := "0 · Scratch: " + opts.ScratchCwd
			workspaceLabels = append(workspaceLabels, label)
			workspaceChoices[label] = ""
			if draft.WorkspaceID == "" {
				workspaceSelected = label
			}
		}
		for i, w := range opts.Workspaces {
			label := fmt.Sprintf("%d · %s · id=%s · path=%s", i+1, w.Title, w.ID, w.Path)
			workspaceLabels = append(workspaceLabels, label)
			workspaceChoices[label] = w.ID
			if draft.WorkspaceID == w.ID {
				workspaceSelected = label
			}
		}
		if _, ok := workspaceChoices[workspaceSelected]; !ok {
			workspaceLabels = append(workspaceLabels, workspaceSelected)
		}
		ui.Select(c.Key("bot-route-workspace"), &workspaceSelected, workspaceLabels).FillWidth().Disabled(!editable || !opts.WorkspacesKnown || !opts.SessionKnown || opts.WorkspaceLocked).OnChange(func() {
			if v, ok := workspaceChoices[workspaceSelected]; ok && m.OnChange != nil {
				draft.WorkspaceID = v
				m.OnChange(draft)
			}
		})
		if !opts.WorkspacesKnown {
			ui.Text(c, "真实 Host 目录选项未核实；目录更改禁用 / Host directories unavailable").TextColor(accent)
		}
		if !opts.SessionKnown || opts.WorkspaceLocked {
			ui.Text(c, "workspaceLocked：存在消息或历史未知，不能改工作区；请在原入口新建 Bot / History present or unknown").TextColor(accent)
		}
		if ui.Button(c, "丢弃模型/目录草稿 / Discard route draft").Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, "审阅模型与目录 / Review model and workspace").Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, "采用匹配的读回资料 / Adopt matching readback").Disabled(blocked).Clicked() && m.OnAdopt != nil {
			m.OnAdopt()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		cmd := r.Command
		old, _ := pluginrpc.DshbotRouteValue(cmd.Expected)
		ui.Column(c.Key("bot-route-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, "确认模型与工作区 / Confirm model and workspace").Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s · canonical Session: %s · revision %d\n原完整模型 / Before: %s\n新完整模型 / After: %s\n原目录 / Before workspace: %s\n新目录 / After workspace: %s\nHost default at review: %s", m.HostLabel, cmd.Expected.Bot.ID, cmd.Expected.Bot.SessionID, cmd.Expected.Revision, botRouteModelText(old.Model), botRouteModelText(cmd.Values.Model), botRouteWorkspaceText(old.WorkspaceID, cmd.Expected.Options), botRouteWorkspaceText(cmd.Values.WorkspaceID, cmd.Expected.Options), botRouteModelText(cmd.Expected.Options.Default))).FontSize(12).TextColor(fg).Selectable()
			if old.WorkspaceID != cmd.Values.WorkspaceID {
				ui.Text(c, "重要：工作区改变会由 Host 创建新的 canonical sessionId 和新会话。旧会话不迁移、不删除；旧会话展示清理可能失败并留下警告。新 ID 只在真实回执后核实；不预造或冒认。blank=true 只是前置保护，Host 的历史检查仍权威，目录 revision CAS 不原子锁定消息历史。").FontSize(12).Bold().TextColor(accent)
			}
			ui.Text(c, "模型/推理强度可能改变后续模型调用、能力及费用；空模型配置沿用 Host 默认值，后续默认值变化仍会影响此 Bot。保存不会取消在途模型或工具，也不撤销已发生的操作。").FontSize(12).TextColor(accent)
			ui.Text(c, "bot/update 会校验模型、写审计、更新会话展示并尝试重命名，还会 requestSync：live idle 会话现在或下次 idle 可能创建目标、重建已暂停目标或修改 active 目标，调用模型、执行工具、发送外部消息并产生费用。冷会话不会在此保存路径解析，之后变为 live/idle 时可能同步；同步失败可能仅记 Host 日志。").FontSize(12).TextColor(accent)
			ui.Text(c, "仅替换 model 与 workspaceId，完整其他资料原样保留；无注册目录时额外携带审阅的 Host scratchCwd。完整回执不证明目标同步或任务完成。此页不主动 prompt/start/resume/open；保存后可刷新并从现有入口打开。关闭界面不撤回已发送请求。未知结果先读回，不自动重试。").FontSize(11).TextColor(muted)
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botRouteAcknowledgement).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, "取消模型/目录保存 / Cancel route save").Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, "确认保存模型/目录 / Confirm route save").Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
