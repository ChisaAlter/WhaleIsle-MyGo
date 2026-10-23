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
	ui.Text(c, i18n.T(KeyPass4PresetsTitle)).FontSize(fonttheme.InterfaceSize(18)).Bold().TextColor(fg)
	ui.Text(c, i18n.T(KeyPass4PresetsHint)).FillWidth().FontSize(fonttheme.InterfaceSize(13)).FixedLineHeight(fonttheme.InterfaceSize(20)).TextColor(muted)
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
			title := i18n.T(KeyPass4Custom)
			if builtIn {
				title = i18n.T(KeyPass4BuiltIn)
			}
			ui.Text(c, title).FontSize(fonttheme.InterfaceSize(13)).TextColor(muted)
			grid := ui.Row(c.Key("preset-grid/" + title)).FillWidth().MinWidth(0).Wrap().AlignItems(ui.Start).Gap(12)
			width := grid.Bounds().W
			grid.Draw(func(_ *ui.Painter, bounds ui.Rect) {
				if bounds.W != width {
					c.Invalidate()
				}
			})
			// Two cards per row at the reference width, one on narrow panes.
			columns := 2
			if width < 480 {
				columns = 1
			}
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
					card := ui.Column(c.Key("preset-row/"+row.ID)).Width(cardWidth).Height(170).MinWidth(0).Gap(0).
						Radius(12).Border(1, ui.RGBA(muted.R, muted.G, muted.B, .18)).Background(c.Theme().Background)
					card.Children(func() {
						main := ui.Button(c, "").Label(providerCopy("选择默认", "Choose default") + " · " + label).MinHeight(82).FillWidth().MinWidth(0).Padding(14).BorderWidth(0).Background(ui.Transparent).Disabled(a.Selected == nil || !roster.Loaded || !roster.Writable || roster.Busy || row.Hidden || row.Broken != "")
						main.Children(func() {
							ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
								ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
									ui.Text(c, label).Shrink(0).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
									badge := i18n.T(KeyPass4BadgeBuiltIn)
									filled := row.ID == roster.DefaultID
									if filled {
										badge = i18n.T(KeyPass4TaskDefault)
									}
									chip := ui.Row(c).Height(18).Padding(0, 6).Radius(4).AlignItems(ui.Center).Shrink(0)
									if filled {
										chip = chip.Background(fg)
									} else {
										chip = chip.Border(.5, ui.RGBA(muted.R, muted.G, muted.B, .5))
									}
									chip.Children(func() {
										ink := muted
										if filled {
											ink = c.Theme().Background
										}
										ui.Text(c, badge).FontSize(fonttheme.InterfaceSize(10)).TextColor(ink)
									})
									ui.Box(c).Grow(1).MinWidth(0)
									ui.Text(c, row.ID).Shrink(0).FontSize(fonttheme.InterfaceSize(10)).
										Font(fonttheme.RenderPreferences().Code.Family).TextColor(muted)
								})
								if description != "" {
									ui.Text(c, description).FillWidth().MaxLines(3).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted).Tooltip(description)
								}
								if row.Broken != "" {
									ui.Text(c, providerCopy("加载失败：", "Failed to load: ")+row.Broken).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
								}
							})
						})
						readerClicked := false
						ui.Box(c).FillWidth().Height(.5).Background(ui.RGBA(muted.R, muted.G, muted.B, .18))
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
				if count == 0 && builtIn {
					ui.Text(c, i18n.T(i18n.KeySettingsAgentPresetsEmpty)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				}
				if !builtIn {
					// The reference's dashed entry: the Agent drafts a new preset
					// mode rather than asking for hand-written JSON.
					label := i18n.T(KeyPass4CreateWithAgent)
					create := ui.Button(c, "").Label(label).FillWidth().Height(46).Radius(12).
						Border(1, ui.RGBA(muted.R, muted.G, muted.B, .5)).BorderStyle(ui.BorderDashed).
						Background(ui.Transparent).TextColor(fg)
					create.Children(func() {
						ui.Text(c, label).FontSize(fonttheme.InterfaceSize(14)).TextColor(fg)
					})
					if create.Clicked() && a.Creator != nil && a.Creator.OnPrepare != nil {
						a.Creator.OnPrepare()
					}
				}
			})
		}
	})
}
// presetBookIcon is the handbook glyph at the right edge of a preset card's
// action row; it opens the same guide view as the text actions.
var presetBookIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8 4.1C6.9 3.3 5.4 3 2.8 3v8.6c2.6 0 4.1.3 5.2 1.1 1.1-.8 2.6-1.1 5.2-1.1V3c-2.6 0-4.1.3-5.2 1.1z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/><path d="M8 4.1v8.6" stroke="currentColor" stroke-width="1.2"/></svg>`))
