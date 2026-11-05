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
	OnSaveDefault         func(string)
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

func agentPresetCardName(row session.ManagedAgentPreset) string {
	name, _ := agentPresetDisplay(row)
	if name == "" {
		return row.ID
	}
	return name
}

func agentPresetsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.AgentPresets == nil || m.AgentPresets.Manager == nil {
		return
	}
	a := *m.AgentPresets
	if a.Reader != nil {
		reader := *a.Reader
		reader.focus = ui.Local(ui.Box(c.Key("preset-reader-focus")).Absolute().Size(0, 0).PassThrough(), "focus", func() presetReaderFocus { return presetReaderFocus{} })
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
	// The 0.3.13 section is a 12 DIP column: an 18/600 title, a 13 DIP intro and
	// the card groups.
	ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
		ui.Text(c, i18n.T(KeyPass4PresetsTitle)).FontSize(fonttheme.InterfaceSize(18)).FixedLineHeight(fonttheme.InterfaceSize(24)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(KeyPass4PresetsHint)).FillWidth().FontSize(fonttheme.InterfaceSize(13)).FixedLineHeight(fonttheme.InterfaceSize(17.3)).TextColor(muted)
		settingsNotice(c, "agent_presets:77", "error", snap.Err != "", func() string { return snap.Err })
		settingsNotice(c, "agent_presets:84", "", snap.Status != "", func() string { return snap.Status })

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
		if snap.Loaded && !snap.Writable {
			ui.Text(c, i18n.T(i18n.KeySettingsReadOnly)).TextColor(muted)
		}
		agentPresetRosterPane(c, &a, snap, fg, muted)
		// The presets reload when the page opens, so there is no refresh button. The
		// Creator block is the second step of the dashed "create with Agent" entry and
		// only shows once that flow has something to say.
		if a.Creator != nil && (a.Creator.State.Review || a.Creator.State.Attempted || a.Creator.State.Error != "") {
			creatorTaskPane(c, a.Creator, snap, fg, muted)
		}
		agentPresetReaderPane(c, a.Reader, snap, fg, muted)
	})
}

func agentPresetRosterPane(c *ui.Context, a *AgentPresetsAccess, roster session.AgentPresetManagerState, fg, muted ui.Color) {
	colors := modelColors(c)
	secondary := fonttheme.Color(colors.Secondary)
	selectorFill := fonttheme.Color(colors.SelectorFill)
	cardBorder := fonttheme.Color(colors.BorderL4)
	ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
		for _, builtIn := range []bool{true, false} {
			title := i18n.T(KeyPass4Custom)
			if builtIn {
				title = i18n.T(KeyPass4BuiltIn)
			}
			// A group is a 10 DIP column under a 12/600 tracked head; the custom
			// group sits 20 DIP further down.
			group := ui.Column(c.Key("preset-group/" + title)).FillWidth().MinWidth(0).Gap(10)
			if !builtIn {
				group = group.Padding(20, 0, 0, 0)
			}
			group.Children(func() {
				ui.Text(c, title).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(16)).FontWeight(600).LetterSpacing(.72).TextColor(muted)
				grid := ui.Row(c.Key("preset-grid/" + title)).FillWidth().MinWidth(0).Wrap().AlignItems(ui.Start).Gap(12)
				width := grid.Bounds().W
				grid.Draw(func(_ *ui.Painter, bounds ui.Rect) {
					if bounds.W != width {
						c.Invalidate()
					}
				})
				// Two cards per row at the reference width, one on narrow panes.
				columns := 2
				if width < 548 {
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
						active := row.ID == roster.DefaultID
						// The default preset is the raised card: the selector fill and
						// the tertiary ink as its 0.667 stroke. The others are the
						// panel layer at 80% under the l4 stroke.
						card := ui.Column(c.Key("preset-row/" + row.ID)).Width(cardWidth).MinWidth(0).Gap(0).Radius(20)
						if active {
							card = card.Border(.667, fonttheme.Color(colors.Muted)).Background(selectorFill)
						} else {
							card = card.Border(.667, cardBorder).Background(modelWell(c).Alpha(.8))
						}
						card.Children(func() {
							main := ui.Button(c, "").Label(providerCopy("设为新任务默认", "Set as new task default")+" · "+label).FillWidth().MinWidth(0).Padding(14, 16, 12, 16).BorderWidth(0).Background(ui.Transparent).Disabled(a.Selected == nil || !roster.Loaded || !roster.Writable || roster.Busy || row.Hidden || row.Broken != "")
							main.Children(func() {
								ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
									ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Start).Children(func() {
										ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
											ui.Text(c, agentPresetCardName(row)).MinWidth(0).SingleLine().Tooltip(label).FontSize(fonttheme.InterfaceSize(15)).FixedLineHeight(fonttheme.InterfaceSize(21)).FontWeight(600).TextColor(fg)
											badge := i18n.T(KeyPass4Custom)
											if row.BuiltIn {
												badge = i18n.T(KeyPass4BadgeBuiltIn)
											}
											if active {
												badge = i18n.T(KeyPass4TaskDefault)
											}
											chip := ui.Row(c).Padding(1, 8).Radius(10).AlignItems(ui.Center).Shrink(0)
											ink, weight := muted, 400
											if active {
												chip = chip.Background(fg)
												ink, weight = selectorFill, 500
											} else {
												chip = chip.Border(.667, cardBorder)
											}
											chip.Children(func() {
												ui.Text(c, badge).FontSize(fonttheme.InterfaceSize(11)).FixedLineHeight(fonttheme.InterfaceSize(17)).FontWeight(weight).TextColor(ink)
											})
										})
										ui.Text(c, row.ID).MaxWidth(max(0, (cardWidth-32)*.45)).MinWidth(0).SingleLine().Tooltip(row.ID).FontSize(fonttheme.InterfaceSize(11)).FixedLineHeight(fonttheme.InterfaceSize(21)).
											Font(fonttheme.RenderPreferences().Code.Family).TextColor(muted)
									})
									if description != "" {
										ui.Text(c, description).FillWidth().MaxLines(3).FontSize(fonttheme.InterfaceSize(13)).FixedLineHeight(fonttheme.InterfaceSize(20.15)).TextColor(secondary).Tooltip(description)
									}
									if row.Broken != "" {
										ui.Text(c, providerCopy("加载失败：", "Failed to load: ")+row.Broken).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
									}
								})
							})
							readerClicked := false
							ui.Box(c).FillWidth().Height(.667).Background(fonttheme.Color(colors.Border))
							ui.Row(c).FillWidth().MinWidth(0).Gap(2).Padding(6, 10).AlignItems(ui.Center).Children(func() {
								ui.Box(c).Grow(1).MinWidth(0).Children(func() { agentPresetGuidePane(c, row, fg, muted) })
								if a.Reader != nil && agentPresetViewAction(c, a.Reader, label, !roster.Loaded || a.Reader.State.Busy || a.Reader.OnSelect == nil) && a.Reader.OnSelect != nil {
									readerClicked = true
									a.Reader.OnSelect(a.Reader.State.Ticket, row.ID)
								}
							})
							if !readerClicked && main.Clicked() && a.Selected != nil {
								*a.Selected = label
								if a.OnSaveDefault != nil {
									a.OnSaveDefault(row.ID)
								} else if a.Manager != nil {
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
							Border(1, ui.RGBA(muted.R, muted.G, muted.B, .22)).BorderStyle(ui.BorderDashed).
							Background(ui.Transparent).TextColor(fg)
						create.Children(func() {
							ui.Text(c, label).FontSize(fonttheme.InterfaceSize(14)).TextColor(fg)
						})
						if create.Clicked() && a.Creator != nil && a.Creator.OnPrepare != nil {
							a.Creator.OnPrepare()
						}
					}
				})
			})
		}
	})
}

// presetBookIcon is the handbook glyph at the right edge of a preset card's
// action row; it opens the same guide view as the text actions.
var presetBookIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8 4.1C6.9 3.3 5.4 3 2.8 3v8.6c2.6 0 4.1.3 5.2 1.1 1.1-.8 2.6-1.1 5.2-1.1V3c-2.6 0-4.1.3-5.2 1.1z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/><path d="M8 4.1v8.6" stroke="currentColor" stroke-width="1.2"/></svg>`))

// presetDocIcon is the outlined document glyph of the 0.3.13 "view
// configuration" button (a 12 x 13.3 outline inside the 16 DIP box).
var presetDocIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M4.4 1.7h7.2a1.1 1.1 0 0 1 1.1 1.1v10.4a1.1 1.1 0 0 1-1.1 1.1H4.4a1.1 1.1 0 0 1-1.1-1.1V2.8a1.1 1.1 0 0 1 1.1-1.1z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/><path d="M5.6 5.4h4.8M5.6 8.4h3" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>`))
