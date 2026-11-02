package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

type ContactsModel struct {
	State      dshbotstore.ContactSnapshot
	Activity   map[string]dshbotstore.ContactActivity
	OnMarkRead func(id string, throughSeq int)
	OnRefresh  func()
	OnChange   func(field, value string)
	OnAction   func(pluginrpc.DshbotContactCommand)
}

func contactsView(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	cm := m.Contacts
	s := cm.State
	blocked := m.Busy || m.PluginRPCBusy || s.Busy || !m.DshbotEnabled || !m.HarnessReady
	readonly := blocked || !s.Loaded || !s.Writable || s.Pending != nil
	change := func(k, v string) {
		if cm.OnChange != nil {
			cm.OnChange(k, v)
		}
	}
	action := func(cmd pluginrpc.DshbotContactCommand) {
		if cm.OnAction != nil {
			cm.OnAction(cmd)
		}
	}
	root := ui.Column(c.Key("bot-contacts")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
	chrome.GlassSurface(root, field, chrome.MaterialPanel, 12)
	root.Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantContacts)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg).Grow(1)
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefresh)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || s.Scope == "").Clicked() && cm.OnRefresh != nil {
				cm.OnRefresh()
			}
		})
		if s.Error != "" {
			ui.Text(c, s.Error).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		query := s.Filters.Query
		ui.TextInput(c.Key("contacts-search"), &query).Label(i18n.T(i18n.KeyMiscAssistantSearchContacts)).FillWidth().Height(32).Radius(8).Border(.5, border).Background(field).Padding(0, 10).OnChange(func() { change("query", query) })
		ui.Row(c).Gap(6).Wrap().Children(func() {
			for _, opt := range []struct{ id, label string }{{"", i18n.T(i18n.KeyW4AssistantAll)}, {"bots", "Bot"}, {"rooms", i18n.T(i18n.KeyW4AssistantGroup)}} {
				opt := opt
				if ui.Button(c, opt.label).Height(28).Padding(0, 10).Radius(999).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(s.Filters.Kind == opt.id).Clicked() {
					change("kind", opt.id)
				}
			}
			hidden := s.Filters.ShowHidden
			if ui.Checkbox(c, &hidden, i18n.T(i18n.KeyW4AssistantShowHiddenItems)).Changed() {
				change("hidden", fmt.Sprint(hidden))
			}
			if ui.Button(c, i18n.T(i18n.KeyW4AssistantClearFilters)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Clicked() {
				change("reset", "")
			}
		})
		ui.Row(c).Gap(6).Wrap().Children(func() {
			for _, opt := range []struct{ id, label string }{{"", i18n.T(i18n.KeyW4AssistantAllActivity)}, {"active", i18n.T(i18n.KeyW4AssistantActive)}, {"recent", i18n.T(i18n.KeyW4AssistantLast7Days)}, {"idle", i18n.T(i18n.KeyW4AssistantIdle)}} {
				opt := opt
				if ui.Button(c, opt.label).Height(28).Padding(0, 10).Radius(999).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(s.Filters.Activity == opt.id).Clicked() {
					change("activity", opt.id)
				}
			}
		})
		sources := dshbotstore.ContactSources(s.Catalog)
		if len(sources) > 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantSource)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Column(c).Gap(4).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantAllSources)).Height(28).Padding(0, 10).Radius(999).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(s.Filters.Source == "").Clicked() {
					change("source", "")
				}
				for _, source := range sources {
					source := source
					if ui.Button(c, source.Label+" · "+source.ID).Height(28).Padding(0, 10).Radius(999).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(s.Filters.Source == source.ID).Clicked() {
						change("source", source.ID)
					}
				}
			})
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantReadingTheCurrentHostSFullContactCata)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyW4AssistantCatalogRevisionDDContactsDSections), s.Catalog.Revision, len(s.Catalog.Items), len(s.Catalog.Sections))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheHostIsReadOnlyRightNowButConversa)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
		if s.Pending != nil && !s.Busy {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantACatalogOperationIsAwaitingVerification)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
		}
		ui.Column(c.Key("contacts-section-editor")).Background(field).Border(.5, border).Radius(12).Padding(10, 12).Gap(6).Children(func() {
			label := i18n.T(i18n.KeyMiscAssistantNewSection)
			if s.EditingSection != "" {
				label = i18n.T(i18n.KeyMiscAssistantRenameSection)
			}
			name := s.SectionName
			ui.TextInput(c.Key("contacts-section-name"), &name).Label(label+" (1–120)").FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(readonly).OnChange(func() { change("name", name) })
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantSaveSection)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly || name == "").Clicked() {
					kind := "create"
					if s.EditingSection != "" {
						kind = "rename"
					}
					action(pluginrpc.DshbotContactCommand{Action: kind, ID: s.EditingSection, Name: name})
				}
				if s.EditingSection != "" && ui.Button(c, i18n.T(i18n.KeyW4AssistantCancelRename)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
					change("edit", "")
				}
			})
		})
		if s.Undo != nil {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyW4AssistantDeletedSectionSItsContactsAreKeptAnd), s.Undo.Section.Name)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRestoreSection)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly).Clicked() {
				action(pluginrpc.DshbotContactCommand{Action: "restore", Snapshot: s.Undo})
			}
		}
		groups := dshbotstore.ContactGroups(s.Catalog, s.Filters, cm.Activity)
		count := 0
		for index, group := range groups {
			index, group := index, group
			grp := ui.Column(c.Key("contact-group-"+group.ID)).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
			chrome.GlassSurface(grp, field, chrome.MaterialPanel, 12)
			grp.Children(func() {
				ui.Text(c, fmt.Sprintf("%s (%d)", group.Name, len(group.Items))).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
				if group.ID != "" {
					ui.Row(c).Gap(6).Wrap().Children(func() {
						if ui.Button(c, i18n.T(i18n.KeyW4AssistantRename)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly).Clicked() {
							change("edit", group.ID)
						}
						if ui.Button(c, i18n.T(i18n.KeyW4AssistantMoveUp)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly || index == 0).Clicked() {
							action(pluginrpc.DshbotContactCommand{Action: "move", ID: group.ID, Delta: -1})
						}
						if ui.Button(c, i18n.T(i18n.KeyW4AssistantMoveDown)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly || index >= len(s.Catalog.Sections)-1).Clicked() {
							action(pluginrpc.DshbotContactCommand{Action: "move", ID: group.ID, Delta: 1})
						}
						if ui.Button(c, i18n.T(i18n.KeyW4AssistantDeleteSection)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly).Clicked() {
							change("delete", group.ID)
						}
					})
					if s.DeleteSection == group.ID {
						ui.Text(c, i18n.T(i18n.KeyW4AssistantOnlyTheSectionIsDeletedAndEveryMember)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
						ui.Row(c).Gap(6).Children(func() {
							if ui.Button(c, i18n.T(i18n.KeyW4AssistantConfirmSectionDeletion)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly).Clicked() {
								action(pluginrpc.DshbotContactCommand{Action: "delete", ID: group.ID})
							}
							if ui.Button(c, i18n.T(i18n.KeyW4AssistantCancel)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
								change("delete", "")
							}
						})
					}
				}
				for _, row := range group.Items {
					row := row
					count++
					item := ui.Column(c.Key("contact-"+row.ID)).Border(.5, border).Radius(12).Padding(10, 12).Gap(5)
					chrome.GlassSurface(item, field, chrome.MaterialPanel, 12)
					item.Children(func() {
						title := dshbotstore.ContactDisplayName(row)
						if row.Pinned {
							title = "★ " + title
						}
						if row.Hidden {
							title += i18n.T(i18n.KeyW4AssistantHidden)
						}
						if row.ID == m.DshbotSelectedID {
							title = "• " + title
						}
						kind := "Bot"
						if row.Kind == "room" {
							kind = i18n.T(i18n.KeyW4AssistantGroup)
						}
						ui.Text(c, kind+" · "+title).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
						if title != row.Name && row.Kind != "room" {
							ui.Text(c, "@"+row.Name).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
						}
						if description := pluginrpc.DshbotContactText(row, "description"); description != "" {
							ui.Text(c, description).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted).Selectable()
						}
						if _, label := pluginrpc.DshbotContactSource(row); label != "" {
							ui.Text(c, i18n.T(i18n.KeyW4AssistantSource)+label).FontSize(fonttheme.InterfaceSize(11)).Padding(2, 8).Radius(999).Border(.5, border).TextColor(muted)
						}
						if live := cm.Activity[row.ID]; live.Active {
							ui.Text(c, i18n.T(i18n.KeyMiscAssistantActive)).FontSize(fonttheme.InterfaceSize(11)).Padding(2, 8).Radius(999).Border(.5, accent).TextColor(accent)
						}
						if live := cm.Activity[row.ID]; live.Unread() {
							ui.Text(c, i18n.T(i18n.KeyMiscAssistantUnreadCompletedResult)).FontSize(fonttheme.InterfaceSize(11)).Padding(2, 8).Radius(999).Border(.5, accent).TextColor(accent)
							if ui.Button(c, i18n.T(i18n.KeyMiscAssistantMarkRead)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly || cm.OnMarkRead == nil).Clicked() {
								cm.OnMarkRead(row.ID, live.ActivitySeq)
							}
						}
						if preview := cm.Activity[row.ID].Preview; preview != "" {
							ui.Text(c, preview).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
						}
						for _, cached := range m.DshbotRoster {
							if cached.ID == row.ID && cached.Detail != "" {
								ui.Text(c, cached.Detail).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
								break
							}
						}
						ui.Row(c).Gap(6).Wrap().Children(func() {
							if m.OnDshbotSelect != nil && ui.Button(c, i18n.T(i18n.KeyW4AssistantSelect)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || row.ID == m.DshbotSelectedID).Clicked() {
								m.OnDshbotSelect(row.ID)
							}
							if m.OnDshbotOpenRosterSession != nil && ui.Button(c, i18n.T(i18n.KeyW4AssistantOpenConversation)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
								m.OnDshbotOpenRosterSession(row.ID)
							}
							label := i18n.T(i18n.KeyW4AssistantPin)
							if row.Pinned {
								label = i18n.T(i18n.KeyW4AssistantUnpin)
							}
							if ui.Button(c, label).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly).Clicked() {
								action(pluginrpc.DshbotContactCommand{Action: "pin", ID: row.ID, SessionID: row.SessionID, Pinned: !row.Pinned})
							}
							if ui.Button(c, i18n.T(i18n.KeyW4AssistantMoveToSection)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly).Clicked() {
								change("assign", row.ID)
							}
						})
						if s.AssignmentItem == row.ID {
							ui.Column(c).Gap(4).Children(func() {
								if ui.Button(c, i18n.T(i18n.KeyW4AssistantMoveToUnassigned)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly || pluginrpc.DshbotContactText(row, "sectionId") == "").Clicked() {
									action(pluginrpc.DshbotContactCommand{Action: "assign", ID: row.ID, SessionID: row.SessionID})
								}
								for _, section := range s.Catalog.Sections {
									section := section
									if ui.Button(c, section.Name).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(readonly || pluginrpc.DshbotContactText(row, "sectionId") == section.ID).Clicked() {
										action(pluginrpc.DshbotContactCommand{Action: "assign", ID: row.ID, SessionID: row.SessionID, SectionID: section.ID})
									}
								}
								if ui.Button(c, i18n.T(i18n.KeyW4AssistantCancelMove)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
									change("assign", "")
								}
							})
						}
					})
				}
			})
		}
		if count == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoMatchingContacts)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		}
	})
}
