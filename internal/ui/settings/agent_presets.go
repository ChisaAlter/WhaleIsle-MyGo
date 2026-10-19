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
	name, _ := agentPresetDisplay(row)
	if row.BuiltIn {
		return name
	}
	if name == "" || name == row.ID {
		return row.ID
	}
	return name + " (" + row.ID + ")"
}

func agentPresetsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.AgentPresets == nil || m.AgentPresets.Manager == nil {
		return
	}
	a := *m.AgentPresets
	if a.Reader != nil {
		reader := *a.Reader
		reader.focus = ui.Local(ui.Box(c.Key("preset-reader-focus")), "focus", func() presetReaderFocus { return presetReaderFocus{} })
		a.Reader = &reader
	}
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
	ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsTitle)).FontSize(fonttheme.InterfaceSize(16)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsHint)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if snap.Err != "" {
		ui.Text(c, snap.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if snap.Status != "" {
		ui.Text(c, snap.Status).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}

	defaultLabel := snap.DefaultID
	for _, row := range snap.Rows {
		label := agentPresetSettingsLabel(row)
		if row.ID == snap.DefaultID {
			defaultLabel = label
		}
		if row.Hidden || row.Broken != "" {
			continue
		}

	}
	if a.Selected != nil && *a.Selected == "" {
		*a.Selected = defaultLabel
	}
	// The persisted value remains visible while an unsaved choice is staged,
	// and on a refusal. Only the manager's confirmed response changes it.
	if snap.Loaded {
		ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsPicker)+": "+defaultLabel).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if !snap.Loaded && snap.Busy {
		ui.Text(c, i18n.T(i18n.KeySettingsLoading)).TextColor(muted)
	}
	if snap.Loaded && !snap.Writable {
		ui.Text(c, i18n.T(i18n.KeySettingsReadOnly)).TextColor(muted)
	}
	agentPresetRosterPane(c, &a, snap, fg, muted)
	ui.Row(c).Gap(8).Wrap().Children(func() {
		if ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Disabled(snap.Busy).Clicked() {
			go a.Manager.Refresh(context.Background())
		}
	})
	creatorTaskPane(c, a.Creator, snap, fg, muted)
	agentPresetReaderPane(c, a.Reader, snap, fg, muted)
}

func agentPresetRosterPane(c *ui.Context, a *AgentPresetsAccess, roster session.AgentPresetManagerState, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
		for _, builtIn := range []bool{true, false} {
			title := "自定义 / Custom"
			if builtIn {
				title = "内置 / Built-in"
			}
			ui.Text(c, title).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(muted)
			grid := ui.Row(c.Key("preset-grid/" + title)).FillWidth().MinWidth(0).Wrap().AlignItems(ui.Start).Gap(12)
			width := grid.Bounds().W
			grid.Draw(func(_ *ui.Painter, bounds ui.Rect) {
				if bounds.W != width {
					c.Invalidate()
				}
			})
			columns := max(1, int((width+12)/280))
			cardWidth := (width - float32(columns-1)*12) / float32(columns)
			grid.Children(func() {
				count := 0
				for _, row := range roster.Rows {
					if row.BuiltIn != builtIn || (row.Hidden && row.Broken == "") {
						continue
					}
					count++
					label := agentPresetSettingsLabel(row)
					_, description := agentPresetDisplay(row)
					card := ui.Column(c.Key("preset-row/"+row.ID)).Width(cardWidth).MinWidth(0).Gap(0).Radius(12).Border(1, ui.RGBA(muted.R, muted.G, muted.B, .20))
					if row.ID == roster.DefaultID {
						card = card.Background(ui.RGBA(muted.R, muted.G, muted.B, .08))
					}
					card.Children(func() {
						main := ui.Button(c, "").Label("选择默认 / Choose default · " + label).MinHeight(82).FillWidth().MinWidth(0).Padding(14).BorderWidth(0).Background(ui.Transparent).Disabled(a.Selected == nil || !roster.Loaded || !roster.Writable || roster.Busy || row.Hidden || row.Broken != "")
						main.Children(func() {
							ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
								ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
									ui.Text(c, label).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
									if row.ID == roster.DefaultID {
										ui.Text(c, "✓").FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
									}
								})
								if description != "" {
									ui.Text(c, description).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).SingleLine().Tooltip(description)
								}
								if row.Broken != "" {
									ui.Text(c, "加载失败 / Failed to load: "+row.Broken).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
								}
							})
						})
						readerClicked := false
						ui.Row(c).FillWidth().MinWidth(0).Gap(4).Padding(4, 10, 10, 10).AlignItems(ui.Center).Children(func() {
							ui.Box(c).Grow(1).MinWidth(0).Children(func() { agentPresetGuidePane(c, row, fg, muted) })
							if a.Reader != nil && agentPresetViewAction(c, a.Reader, label, !roster.Loaded || a.Reader.State.Busy || a.Reader.OnSelect == nil) && a.Reader.OnSelect != nil {
								readerClicked = true
								a.Reader.OnSelect(a.Reader.State.Ticket, row.ID)
							}
						})
						if !readerClicked && main.Clicked() && a.Selected != nil {
							*a.Selected = label
							if a.Manager != nil {
								go a.Manager.SaveDefault(context.Background(), row.ID)
							}
							if a.Reader != nil && a.Reader.OnClose != nil {
								a.Reader.OnClose(a.Reader.State.Ticket)
							}
						}

					})
				}
				if count == 0 {
					ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				}
			})
		}
	})
}
