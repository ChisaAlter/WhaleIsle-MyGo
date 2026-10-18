package settings

import (
	"context"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// AgentPresetsAccess edits the persisted default for future sessions. The
// composer owns current-session agentPresets/select independently.
type AgentPresetsAccess struct {
	DisableDeveloperTools bool
	Manager               *session.AgentPresetManager
	Creator               *CreatorTaskAccess
	Selected              *string
	Reader                *AgentPresetReaderAccess
}

func agentPresetSettingsLabel(row session.ManagedAgentPreset) string {
	if row.Name == "" || row.Name == row.ID {
		return row.ID
	}
	return row.Name + " (" + row.ID + ")"
}

func agentPresetsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.AgentPresets == nil || m.AgentPresets.Manager == nil {
		return
	}
	a := *m.AgentPresets
	snap := a.Manager.Snapshot()
	if a.DisableDeveloperTools {
		rows := make([]session.ManagedAgentPreset, 0, len(snap.Rows))
		for _, row := range snap.Rows {
			if row.RequiresCodingTools {
				if a.Reader != nil && a.Reader.State.SelectedID == row.ID {
					reader := *a.Reader
					reader.State.SelectedID = ""
					reader.State.Open = false
					a.Reader = &reader
				}
				continue
			}
			rows = append(rows, row)
		}
		snap.Rows = rows
	}
	ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsTitle)).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsHint)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if snap.Err != "" {
		ui.Text(c, snap.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if snap.Status != "" {
		ui.Text(c, snap.Status).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	byLabel := map[string]string{}
	defaultLabel := snap.DefaultID
	for _, row := range snap.Rows {
		label := agentPresetSettingsLabel(row)
		if row.ID == snap.DefaultID {
			defaultLabel = label
		}
		if row.Hidden || row.Broken != "" {
			continue
		}
		byLabel[label] = row.ID
	}
	if a.Selected != nil && *a.Selected == "" {
		*a.Selected = defaultLabel
	}
	// The persisted value remains visible while an unsaved choice is staged,
	// and on a refusal. Only the manager's confirmed response changes it.
	if snap.Loaded {
		ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsPicker)+": "+defaultLabel).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	agentPresetRosterPane(c, &a, snap, fg, muted)
	if a.Selected != nil && *a.Selected != defaultLabel {
		ui.Text(c, "待保存默认 / Pending default: "+*a.Selected).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	selectedID := ""
	if a.Selected != nil {
		selectedID = byLabel[*a.Selected]
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Disabled(snap.Busy).Clicked() {
			go a.Manager.Refresh(context.Background())
		}
		if ui.Button(c, i18n.T(i18n.KeySettingsSave)).Disabled(!snap.Loaded || !snap.Writable || snap.Busy || selectedID == "" || selectedID == snap.DefaultID).Clicked() {
			go a.Manager.SaveDefault(context.Background(), selectedID)
		}
	})
	creatorTaskPane(c, a.Creator, snap, fg, muted)
	agentPresetReaderPane(c, a.Reader, snap, fg, muted)
}

func agentPresetRosterPane(c *ui.Context, a *AgentPresetsAccess, roster session.AgentPresetManagerState, fg, muted ui.Color) {
	for _, builtIn := range []bool{true, false} {
		title := "自定义 / Custom"
		if builtIn {
			title = "内置 / Built-in"
		}
		ui.Column(c.Key(title)).Gap(7).Children(func() {
			ui.Text(c, title).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
			count := 0
			for _, row := range roster.Rows {
				if row.BuiltIn != builtIn || (row.Hidden && row.Broken == "") {
					continue
				}
				count++
				ui.Column(c.Key("preset-row-" + row.ID)).Gap(5).Children(func() {
					label := agentPresetSettingsLabel(row)
					ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
					if row.Description != "" {
						ui.Text(c, row.Description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
					}
					if row.ID == roster.DefaultID {
						ui.Text(c, "新任务默认 / New task default").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					}
					if row.Broken != "" {
						ui.Text(c, "加载失败 / Failed to load: "+row.Broken).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
					}
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, "选择默认 / Choose default · "+label).Disabled(a.Selected == nil || !roster.Loaded || !roster.Writable || roster.Busy || row.Hidden || row.Broken != "").Clicked() && a.Selected != nil {
							*a.Selected = label
							if a.Reader != nil && a.Reader.OnClose != nil {
								a.Reader.OnClose(a.Reader.State.Ticket)
							}
						}
						if a.Reader != nil && ui.Button(c, "选择阅读 / Select for reading · "+label).Disabled(!roster.Loaded || a.Reader.OnSelect == nil).Clicked() && a.Reader.OnSelect != nil {
							a.Reader.OnSelect(a.Reader.State.Ticket, row.ID)
						}
					})
					agentPresetGuidePane(c, row, fg, muted)
				})
			}
			if count == 0 {
				ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsEmpty)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		})
	}
}
