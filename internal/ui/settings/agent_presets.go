package settings

import (
	"context"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// AgentPresetsAccess edits the persisted default for future sessions. The
// composer owns current-session agentPresets/select independently.
type AgentPresetsAccess struct {
	Manager    *session.AgentPresetManager
	Selected   *string
	DocPreview string
	OnRead     func(id string)
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
	a := m.AgentPresets
	snap := a.Manager.Snapshot()
	ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsHint)).FontSize(12).TextColor(muted)
	if snap.Err != "" {
		ui.Text(c, snap.Err).FontSize(12).TextColor(fg)
	}
	if snap.Status != "" {
		ui.Text(c, snap.Status).FontSize(11).TextColor(muted)
	}
	options := []string{}
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
		options = append(options, label)
		byLabel[label] = row.ID
	}
	if a.Selected != nil && *a.Selected == "" {
		*a.Selected = defaultLabel
	}
	// The persisted value remains visible while an unsaved choice is staged,
	// and on a refusal. Only the manager's confirmed response changes it.
	if snap.Loaded {
		ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsPicker)+": "+defaultLabel).FontSize(11).TextColor(muted)
	}
	desktopRow(c, i18n.T(i18n.KeySettingsAgentPresetsPicker), func() {
		if a.Selected == nil {
			return
		}
		if len(options) == 0 {
			ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsEmpty)).FontSize(11).TextColor(muted)
			return
		}
		ui.Select(c, a.Selected, options).FillWidth().Disabled(snap.Busy || !snap.Writable)
	}, fg, muted)
	selectedID := ""
	if a.Selected != nil {
		selectedID = byLabel[*a.Selected]
	}
	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Disabled(snap.Busy).Clicked() {
			go a.Manager.Refresh(context.Background())
		}
		if a.OnRead != nil && ui.Button(c, i18n.T(i18n.KeySettingsAgentPresetsRead)).Disabled(selectedID == "" || snap.Busy).Clicked() {
			a.OnRead(selectedID)
		}
		if ui.Button(c, i18n.T(i18n.KeySettingsSave)).Disabled(!snap.Loaded || !snap.Writable || snap.Busy || selectedID == "" || selectedID == snap.DefaultID).Clicked() {
			go a.Manager.SaveDefault(context.Background(), selectedID)
		}
	})
	if a.DocPreview != "" {
		ui.Text(c, a.DocPreview).FontSize(11).TextColor(muted)
	}
}
