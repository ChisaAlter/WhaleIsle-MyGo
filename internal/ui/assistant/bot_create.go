package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strconv"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botCreateAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandBotAndSessionEffects)
}
func botCreateConfirmLabel() string {
	return i18n.T(i18n.KeyMiscAssistantConfirmCreateAndOpenBot)
}

type BotCreateModel struct {
	State                                                                 dshbotstore.BotCreateSnapshot
	HostLabel                                                             string
	ModelOptions                                                          []session.ModelOption
	ModelsKnown                                                           bool
	Workspaces                                                            []session.WorkspaceRow
	WorkspacesKnown                                                       bool
	OnSelectModel                                                         func(pluginrpc.DshbotRouteModel)
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
	if d.Provider == "" {
		d.Provider = i18n.T(i18n.KeyMiscAssistantApplicationDefault)
		d.Model = i18n.T(i18n.KeyMiscAssistantResolvedByHost)
	}
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantBotCreateConfig), host, cmd.Expected.Revision, d.Name, d.Title, d.Description, d.WorkspaceID, d.ScratchCwd, cmd.ID, cmd.SessionID, d.Provider, d.Model, d.ReasoningEffort, d.MemoryReview, d.Notify)
}
func botCreateView(c *ui.Context, m BotCreateModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	ui.Column(c.Key("native-bot-create")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCreatePersonalBot)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCreateABotWithTheApplicationDefaul)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if m.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshBotCreationCatalog)).Disabled(blocked || s.Scope == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadCurrentCatalogAndAuthorityFirst)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		} else if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable3)).TextColor(accent)
		}
		if s.Created.ID != "" {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantCreatedDetail), s.Created.Name, s.Created.ID, s.Created.SessionID)).TextColor(fg).Selectable()
			ui.Text(c, "打开仅核实或恢复这个 Bot 的专属会话。打开失败会保留创建结果，不重新创建。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if m.OnOpen != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantOpenCreatedBot)).Disabled(blocked).Clicked() {
				m.OnOpen()
			}
		}
		if s.Draft == nil {
			if m.OnStart != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantNewPersonalBot)).Disabled(blocked || !s.Loaded || !s.Writable || s.Uncertain).Clicked() {
				m.OnStart()
			}
			return
		}
		d := *s.Draft
		disabled := blocked || s.Uncertain || s.Review != nil
		ui.Text(c, fmt.Sprintf("Host: %s · revision %d", m.HostLabel, s.Catalog.Revision)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		for _, f := range []struct{ id, label, value string }{{"name", i18n.T(i18n.KeyMiscAssistantStableBotName120), d.Name}, {"title", i18n.T(i18n.KeyMiscAssistantDisplayTitle120), d.Title}, {"scratchCwd", i18n.T(i18n.KeyMiscAssistantScratchCwdRequired), d.ScratchCwd}} {
			f := f
			value := f.value
			ui.TextInput(c.Key("new-bot-"+f.id), &value).Label(f.label).FillWidth().Disabled(disabled || f.id == "scratchCwd" && d.WorkspaceID != "").OnChange(func() {
				if m.OnChange != nil {
					m.OnChange(f.id, value)
				}
			})
		}
		createWorkspacePicker(c.Key("new-bot-workspace"), d.WorkspaceID, m.Workspaces, m.WorkspacesKnown, disabled, m.OnChange, muted)
		botCreateModelPicker(c, m, d, disabled, muted)
		description := d.Description
		ui.TextArea(c.Key("new-bot-description"), &description).Label(i18n.T(i18n.KeyMiscAssistantDescription12000)).Lines(3, 8).FillWidth().Disabled(disabled).OnChange(func() {
			if m.OnChange != nil {
				m.OnChange("description", description)
			}
		})
		for _, f := range []struct {
			id, label string
			value     bool
		}{{"memoryReview", i18n.T(i18n.KeyMiscAssistantMemoryReview), d.MemoryReview}, {"notify", i18n.T(i18n.KeyMiscAssistantNotifications), d.Notify}} {
			f := f
			value := f.value
			if ui.Checkbox(c.Key("new-bot-"+f.id), &value, f.label).Disabled(disabled).Changed() && m.OnChange != nil {
				m.OnChange(f.id, strconv.FormatBool(value))
			}
		}
		if s.Uncertain {
			ui.Text(c, "创建结果未知：保留并锁定草稿与身份，只读核对，不自动重试。没有记录也不能重建。关闭或隐藏界面不取消已发送创建；保护仅保留在当前应用进程。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
			if row := s.Candidate; row != nil {
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantExactMatchingRecord), row.Name, row.ID, row.SessionID)).TextColor(fg).Selectable()
				if s.Pending != nil {
					ui.Text(c, botCreateReviewText(m.HostLabel, *s.Pending)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
				}
				ui.Text(c, "所有提交字段与该身份记录匹配；目录存在不证明会话创建完成。采用仅保留已有身份，之后单独打开会核实或恢复 canonical 会话。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if m.OnAdopt != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAdoptInspectedBot)).Disabled(blocked).Clicked() {
					m.OnAdopt(row.ID)
				}
			}
		} else {
			if m.OnDiscard != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardBotDraft)).Disabled(blocked).Clicked() {
				m.OnDiscard()
			}
			if m.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewBotCreation)).Disabled(disabled || !s.Loaded || !s.Writable).Clicked() {
				m.OnReview()
			}
		}
		if r := s.Review; r != nil {
			ui.Column(c.Key("new-bot-confirmation")).Border(1, accent).Radius(8).Padding(12).Gap(7).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewNewBot)).Bold().TextColor(fg)
				ui.Text(c, botCreateReviewText(m.HostLabel, r.Command)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, "Host 将校验模型配置、写入 Bot 目录与审计记录、创建并打开专属会话，可能触发会话生命周期处理。非空 workspace 优先于 scratch cwd。后续 Bot 模型、工具、记忆复盘和通知可能产生费用或外部消息。").FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, "本次不发送 prompt、不调用模型生成、不启动 objective 或例行任务、不发送 IM。使用上方固定头像与全部能力；没有 peer 来源和目标。创建未知不自动重试；打开失败与创建失败分开处理。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, botCreateAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if m.OnDismiss != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantBackToBotDraft)).Disabled(blocked).Clicked() {
					m.OnDismiss()
				}
				if m.OnConfirm != nil && ui.Button(c, botCreateConfirmLabel()).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() {
					m.OnConfirm()
				}
			})
		}
	})
}

func createWorkspacePicker(c *ui.Context, workspaceID string, rows []session.WorkspaceRow, known, disabled bool, change func(string, string), muted ui.Color) {
	scratch := i18n.T(i18n.KeyMiscAssistantScratchDirectory)
	labels := []string{scratch}
	choices := map[string]string{scratch: ""}
	selected := scratch
	for i, row := range rows {
		name := row.Title
		if name == "" {
			name = row.Path
		}
		label := fmt.Sprintf("%d · %s · %s", i+1, name, row.Path)
		labels = append(labels, label)
		choices[label] = row.WorkspaceID
		if row.WorkspaceID == workspaceID {
			selected = label
		}
	}
	if workspaceID != "" && selected == scratch {
		selected = i18n.T(i18n.KeyMiscAssistantPreserve) + workspaceID
		labels = append(labels, selected)
	}
	ui.Text(c, i18n.T(i18n.KeyMiscAssistantWorkspace)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Select(c.Key("workspace-choice"), &selected, labels).FillWidth().Disabled(disabled).OnChange(func() {
		if id, ok := choices[selected]; ok && change != nil {
			change("workspaceId", id)
		}
	})
	if !known {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantRegistryNotLoadedKeepTheCurrentWor)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
}

// Choices reuse the current authenticated Host's already loaded picker metadata.
// Route IDs stay opaque; captions never get parsed back into a provider/model.
func botCreateModelPicker(c *ui.Context, m BotCreateModel, d dshbotstore.BotCreateDraft, disabled bool, muted ui.Color) {
	defaultLabel := i18n.T(i18n.KeyMiscAssistantApplicationDefault)
	labels := []string{defaultLabel}
	choices := map[string]pluginrpc.DshbotRouteModel{defaultLabel: {}}
	selected := defaultLabel
	var selectedOption *session.ModelOption
	for i, option := range m.ModelOptions {
		label := fmt.Sprintf("%d · %s", i+1, option.Label)
		labels = append(labels, label)
		choice := pluginrpc.DshbotRouteModel{Provider: option.Provider, Model: option.Model}
		if option.Reasoning != nil && option.Reasoning.DefaultEffort != nil {
			choice.ReasoningEffort = *option.Reasoning.DefaultEffort
		}
		choices[label] = choice
		if option.Provider == d.Provider && option.Model == d.Model {
			selected = label
			copy := option
			selectedOption = &copy
		}
	}
	if d.Provider != "" && selectedOption == nil {
		selected = fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantPreserveModel), d.Provider, d.Model)
		labels = append(labels, selected)
	}
	ui.Text(c, i18n.T(i18n.KeyMiscAssistantModel)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Select(c.Key("new-bot-model-choice"), &selected, labels).FillWidth().Disabled(disabled).OnChange(func() {
		if choice, ok := choices[selected]; ok && m.OnSelectModel != nil {
			m.OnSelectModel(choice)
		}
	})
	if !m.ModelsKnown {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantCatalogNotLoadedKeepTheCurrentChoi)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	effortDefault := i18n.T(i18n.KeyMiscAssistantModelDefaultEffort)
	effortLabels := []string{effortDefault}
	efforts := map[string]string{effortDefault: ""}
	effortSelected := effortDefault
	if selectedOption != nil && selectedOption.Reasoning != nil {
		for i, effort := range selectedOption.Reasoning.Efforts {
			label := fmt.Sprintf("%d · %s", i+1, effort.Name)
			if effort.Name == "" {
				label = fmt.Sprintf("%d · %s", i+1, effort.Id)
			}
			effortLabels = append(effortLabels, label)
			efforts[label] = effort.Id
			if effort.Id == d.ReasoningEffort {
				effortSelected = label
			}
		}
	}
	if d.ReasoningEffort != "" && effortSelected == effortDefault {
		effortSelected = i18n.T(i18n.KeyMiscAssistantPreserve2) + d.ReasoningEffort
		effortLabels = append(effortLabels, effortSelected)
	}
	ui.Text(c, i18n.T(i18n.KeyMiscAssistantReasoningEffort)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	ui.Select(c.Key("new-bot-effort-choice"), &effortSelected, effortLabels).FillWidth().Disabled(disabled || selectedOption == nil || selectedOption.Reasoning == nil).OnChange(func() {
		if effort, ok := efforts[effortSelected]; ok && m.OnSelectModel != nil {
			m.OnSelectModel(pluginrpc.DshbotRouteModel{Provider: d.Provider, Model: d.Model, ReasoningEffort: effort})
		}
	})
}
