package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botRouteAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandModelDirectoryNewSession)
}

type BotRouteModel struct {
	State                                                         dshbotstore.RouteSnapshot
	HostLabel                                                     string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard, OnAdopt func()
	OnChange                                                      func(pluginrpc.DshbotRouteValues)
	OnAcknowledge                                                 func(bool)
}

func botRouteModelText(v pluginrpc.DshbotRouteModel) string {
	if v.Provider == "" {
		return i18n.T(i18n.KeyMiscAssistantHostDefault)
	}
	return fmt.Sprintf("provider=%s · model=%s · effort=%s", v.Provider, v.Model, v.ReasoningEffort)
}
func botRouteWorkspaceText(id string, o pluginrpc.DshbotRouteOptions) string {
	if id == "" {
		return i18n.T(i18n.KeyMiscAssistantScratch) + o.ScratchCwd
	}
	for _, w := range o.Workspaces {
		if w.ID == id {
			return fmt.Sprintf("%s · id=%s · path=%s", w.Title, w.ID, w.Path)
		}
	}
	return "id=" + id + i18n.T(i18n.KeyMiscAssistantPathUnavailable)
}
func botRouteBlock(c *ui.Context, m BotRouteModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && !s.Uncertain && s.Review == nil
	ui.Column(c.Key("bot-route-editor")).Gap(7).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantModelAndWorkspace)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, "选择当前 Host 的模型、推理强度和真实目录。完整资料无法无损保留时，请使用现有 Bot Web 编辑入口。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshModelAndWorkspace)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		for _, w := range s.Warnings {
			ui.Text(c, "Host warning: "+w).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadBackBeforeSaving4)).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		before, err := pluginrpc.DshbotRouteValue(s.Status)
		if err != nil {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantIncompleteProfileUseBotWeb)).TextColor(accent)
			return
		}
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantRouteDetail), m.HostLabel, s.Status.Bot.Name, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision, botRouteModelText(before.Model), botRouteWorkspaceText(before.WorkspaceID, s.Status.Options))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable)).TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		draft := s.Draft
		opts := s.Status.Options
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantModelDraft)+botRouteModelText(draft.Model)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !opts.ModelsKnown {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantModelDiscoveryUnavailable)).TextColor(accent)
		}
		modelLabels := []string{i18n.T(i18n.KeyMiscAssistantHostDefault)}
		modelChoices := map[string]pluginrpc.DshbotRouteModel{modelLabels[0]: {}}
		modelSelected := i18n.T(i18n.KeyMiscAssistantPreserveCurrent) + botRouteModelText(draft.Model)
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
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCurrentHostDefault)+botRouteModelText(opts.Default)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		effortLabels := []string{i18n.T(i18n.KeyMiscAssistantModelDefaultEffort)}
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
			effortSelected = i18n.T(i18n.KeyMiscAssistantPreserve3) + draft.Model.ReasoningEffort
			effortLabels = append(effortLabels, effortSelected)
		}
		ui.Select(c.Key("bot-route-effort"), &effortSelected, effortLabels).FillWidth().Disabled(!editable || !opts.ModelsKnown || draft.Model.Provider == "").OnChange(func() {
			if v, ok := effortChoices[effortSelected]; ok && m.OnChange != nil {
				draft.Model.ReasoningEffort = v
				m.OnChange(draft)
			}
		})
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantWorkspaceDraft)+botRouteWorkspaceText(draft.WorkspaceID, opts)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		workspaceLabels := []string{}
		workspaceChoices := map[string]string{}
		workspaceSelected := i18n.T(i18n.KeyMiscAssistantPreserveCurrent2) + draft.WorkspaceID
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
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantHostDirectoriesUnavailable)).TextColor(accent)
		}
		if !opts.SessionKnown || opts.WorkspaceLocked {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantHistoryPresentOrUnknown)).TextColor(accent)
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardRouteDraft)).Disabled(!editable || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewModelAndWorkspace)).Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Candidate && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAdoptMatchingReadback)).Disabled(blocked).Clicked() && m.OnAdopt != nil {
			m.OnAdopt()
		}
		if s.Review == nil {
			return
		}
		r := s.Review
		cmd := r.Command
		old, _ := pluginrpc.DshbotRouteValue(cmd.Expected)
		ui.Column(c.Key("bot-route-confirmation")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantConfirmModelAndWorkspace)).Bold().TextColor(fg)
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantRouteReviewDetail), m.HostLabel, cmd.Expected.Bot.ID, cmd.Expected.Bot.SessionID, cmd.Expected.Revision, botRouteModelText(old.Model), botRouteModelText(cmd.Values.Model), botRouteWorkspaceText(old.WorkspaceID, cmd.Expected.Options), botRouteWorkspaceText(cmd.Values.WorkspaceID, cmd.Expected.Options), botRouteModelText(cmd.Expected.Options.Default))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			if old.WorkspaceID != cmd.Values.WorkspaceID {
				ui.Text(c, "重要：工作区改变会由 Host 创建新的 canonical sessionId 和新会话。旧会话不迁移、不删除；旧会话展示清理可能失败并留下警告。新 ID 只在真实回执后核实；不预造或冒认。blank=true 只是前置保护，Host 的历史检查仍权威，目录 revision CAS 不原子锁定消息历史。").FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(accent)
			}
			ui.Text(c, "模型/推理强度可能改变后续模型调用、能力及费用；空模型配置沿用 Host 默认值，后续默认值变化仍会影响此 Bot。保存不会取消在途模型或工具，也不撤销已发生的操作。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "bot/update 会校验模型、写审计、更新会话展示并尝试重命名，还会 requestSync：live idle 会话现在或下次 idle 可能创建目标、重建已暂停目标或修改 active 目标，调用模型、执行工具、发送外部消息并产生费用。冷会话不会在此保存路径解析，之后变为 live/idle 时可能同步；同步失败可能仅记 Host 日志。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			ui.Text(c, "仅替换 model 与 workspaceId，完整其他资料原样保留；无注册目录时额外携带审阅的 Host scratchCwd。完整回执不证明目标同步或任务完成。此页不主动 prompt/start/resume/open；保存后可刷新并从现有入口打开。关闭界面不撤回已发送请求。未知结果先读回，不自动重试。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botRouteAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelRouteSave)).Disabled(blocked).Clicked() && m.OnDismiss != nil {
				m.OnDismiss()
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmRouteSave)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
				m.OnConfirm()
			}
		})
	})
}
