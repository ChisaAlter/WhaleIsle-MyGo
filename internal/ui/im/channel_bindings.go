package im

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/imonboard"
	"whaleisle/internal/ui/theme"
)

// Match the existing standalone IM workspace/preset controls on each original
// channel card, using this page's captured channel scope and Bot identity.
func channelBindingControls(c *ui.Context, m Model, v imonboard.View, b imonboard.Bot, disabled bool, fg, muted ui.Color) {
	s := &m.Store.Onboarding
	disabled = disabled || v.Provision.AttemptID != ""
	ui.Text(c, "当前工作区 / Workspace: "+b.Workspace).TextColor(muted).Selectable()
	workspace := b.Workspace
	if draft, ok := v.WorkspaceDrafts[b.BotID]; ok {
		workspace = draft
	}
	if ui.TextInput(c.Key("channel-workspace"), &workspace).Label("Workspace path").FillWidth().Disabled(disabled).Changed() {
		s.EditWorkspace(v.Scope, b.BotID, workspace)
	}
	if ui.Button(c, "保存工作区 / Save workspace").Disabled(disabled || strings.TrimSpace(workspace) == "" || strings.TrimSpace(workspace) == b.Workspace).Clicked() {
		s.SaveWorkspace(v.Scope, b.BotID, workspace)
	}
	inherit := "跟随 Host 默认 / Host default"
	labels := []string{inherit}
	ids := map[string]string{inherit: ""}
	selected := inherit
	available := b.AgentPreset == ""
	for i, preset := range v.Presets {
		label := fmt.Sprintf("%d · %s (%s)", i+1, preset.Label, preset.ID)
		labels = append(labels, label)
		ids[label] = preset.ID
		if b.AgentPreset == preset.ID {
			selected, available = label, true
		}
	}
	if !available {
		selected = b.AgentPreset + " / Unavailable"
		labels = append(labels, selected)
		ids[selected] = b.AgentPreset
		ui.Text(c, "当前 Agent Preset 已不可用，可选择其他 Preset 或跟随 Host 默认。 / Current preset unavailable.").TextColor(muted)
	}
	ui.Text(c, "Agent Preset").TextColor(fg)
	ui.Select(c.Key("channel-preset"), &selected, labels).FillWidth().Disabled(disabled).OnChange(func() {
		if id, ok := ids[selected]; ok && id != b.AgentPreset {
			s.SaveAgentPreset(v.Scope, b.BotID, id)
		}
	})
	ui.Text(c, "仅影响新会话；切换当前聊天请先发送 /new。 / Applies to new sessions; send /new to switch the current chat.").FontSize(theme.InterfaceSize(11)).TextColor(muted)
}
