package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
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
	ui.Column(c.Key("bot-contacts")).Border(1, border).Radius(8).Padding(12).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantContacts)).Bold().TextColor(fg).Grow(1)
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefresh)).Disabled(blocked || s.Scope == "").Clicked() && cm.OnRefresh != nil {
				cm.OnRefresh()
			}
		})
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		if s.Notice != "" {
			ui.Text(c, s.Notice).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
		query := s.Filters.Query
		ui.TextInput(c.Key("contacts-search"), &query).Label(i18n.T(i18n.KeyMiscAssistantSearchContacts)).FillWidth().OnChange(func() { change("query", query) })
		ui.Row(c).Gap(6).Children(func() {
			for _, opt := range []struct{ id, label string }{{"", i18n.T(i18n.KeyW4AssistantAll)}, {"bots", "Bot"}, {"rooms", i18n.T(i18n.KeyW4AssistantGroup)}} {
				opt := opt
				if ui.Button(c, opt.label).Disabled(s.Filters.Kind == opt.id).Clicked() {
					change("kind", opt.id)
				}
			}
			hidden := s.Filters.ShowHidden
			if ui.Checkbox(c, &hidden, i18n.T(i18n.KeyW4AssistantShowHiddenItems)).Changed() {
				change("hidden", fmt.Sprint(hidden))
			}
			if ui.Button(c, i18n.T(i18n.KeyW4AssistantClearFilters)).Clicked() {
				change("reset", "")
			}
		})
		ui.Row(c).Gap(6).Children(func() {
			for _, opt := range []struct{ id, label string }{{"", i18n.T(i18n.KeyW4AssistantAllActivity)}, {"active", i18n.T(i18n.KeyW4AssistantActive)}, {"recent", i18n.T(i18n.KeyW4AssistantLast7Days)}, {"idle", i18n.T(i18n.KeyW4AssistantIdle)}} {
				opt := opt
				if ui.Button(c, opt.label).Disabled(s.Filters.Activity == opt.id).Clicked() {
					change("activity", opt.id)
				}
			}
		})
		sources := dshbotstore.ContactSources(s.Catalog)
		if len(sources) > 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantSource)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Column(c).Gap(4).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantAllSources)).Disabled(s.Filters.Source == "").Clicked() {
					change("source", "")
				}
				for _, source := range sources {
					source := source
					if ui.Button(c, source.Label+" · "+source.ID).Disabled(s.Filters.Source == source.ID).Clicked() {
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
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheHostIsReadOnlyRightNowButConversa)).TextColor(muted)
		}
		if s.Pending != nil && !s.Busy {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantACatalogOperationIsAwaitingVerification)).TextColor(accent)
		}
		ui.Column(c.Key("contacts-section-editor")).Background(field).Padding(8).Gap(6).Children(func() {
			label := i18n.T(i18n.KeyMiscAssistantNewSection)
			if s.EditingSection != "" {
				label = i18n.T(i18n.KeyMiscAssistantRenameSection)
			}
			name := s.SectionName
			ui.TextInput(c.Key("contacts-section-name"), &name).Label(label + " (1–120)").FillWidth().Disabled(readonly).OnChange(func() { change("name", name) })
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantSaveSection)).Disabled(readonly || name == "").Clicked() {
					kind := "create"
					if s.EditingSection != "" {
						kind = "rename"
					}
					action(pluginrpc.DshbotContactCommand{Action: kind, ID: s.EditingSection, Name: name})
				}
				if s.EditingSection != "" && ui.Button(c, i18n.T(i18n.KeyW4AssistantCancelRename)).Disabled(blocked).Clicked() {
					change("edit", "")
				}
			})
		})
		if s.Undo != nil {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyW4AssistantDeletedSectionSItsContactsAreKeptAnd), s.Undo.Section.Name)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRestoreSection)).Disabled(readonly).Clicked() {
				action(pluginrpc.DshbotContactCommand{Action: "restore", Snapshot: s.Undo})
			}
		}
		groups := dshbotstore.ContactGroups(s.Catalog, s.Filters, cm.Activity)
		count := 0
		for index, group := range groups {
			index, group := index, group
			ui.Column(c.Key("contact-group-"+group.ID)).Border(1, border).Radius(6).Padding(10).Gap(6).Children(func() {
				ui.Text(c, fmt.Sprintf("%s (%d)", group.Name, len(group.Items))).Bold().TextColor(fg)
				if group.ID != "" {
					ui.Row(c).Gap(6).Children(func() {
						if ui.Button(c, i18n.T(i18n.KeyW4AssistantRename)).Disabled(readonly).Clicked() {
							change("edit", group.ID)
						}
						if ui.Button(c, i18n.T(i18n.KeyW4AssistantMoveUp)).Disabled(readonly || index == 0).Clicked() {
							action(pluginrpc.DshbotContactCommand{Action: "move", ID: group.ID, Delta: -1})
						}
						if ui.Button(c, i18n.T(i18n.KeyW4AssistantMoveDown)).Disabled(readonly || index >= len(s.Catalog.Sections)-1).Clicked() {
							action(pluginrpc.DshbotContactCommand{Action: "move", ID: group.ID, Delta: 1})
						}
						if ui.Button(c, i18n.T(i18n.KeyW4AssistantDeleteSection)).Disabled(readonly).Clicked() {
							change("delete", group.ID)
						}
					})
					if s.DeleteSection == group.ID {
						ui.Text(c, i18n.T(i18n.KeyW4AssistantOnlyTheSectionIsDeletedAndEveryMember)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
						ui.Row(c).Gap(6).Children(func() {
							if ui.Button(c, i18n.T(i18n.KeyW4AssistantConfirmSectionDeletion)).Disabled(readonly).Clicked() {
								action(pluginrpc.DshbotContactCommand{Action: "delete", ID: group.ID})
							}
							if ui.Button(c, i18n.T(i18n.KeyW4AssistantCancel)).Disabled(blocked).Clicked() {
								change("delete", "")
							}
						})
					}
				}
				for _, row := range group.Items {
					row := row
					count++
					ui.Column(c.Key("contact-" + row.ID)).Background(field).Padding(8).Gap(5).Children(func() {
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
						ui.Text(c, kind+" · "+title).TextColor(fg).Bold()
						if title != row.Name && row.Kind != "room" {
							ui.Text(c, "@"+row.Name).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						}
						if description := pluginrpc.DshbotContactText(row, "description"); description != "" {
							ui.Text(c, description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
						}
						if _, label := pluginrpc.DshbotContactSource(row); label != "" {
							ui.Text(c, i18n.T(i18n.KeyW4AssistantSource)+label).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						}
						if live := cm.Activity[row.ID]; live.Active {
							ui.Text(c, i18n.T(i18n.KeyMiscAssistantActive)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
						}
						if live := cm.Activity[row.ID]; live.Unread() {
							ui.Text(c, i18n.T(i18n.KeyMiscAssistantUnreadCompletedResult)).TextColor(accent)
							if ui.Button(c, i18n.T(i18n.KeyMiscAssistantMarkRead)).Disabled(readonly || cm.OnMarkRead == nil).Clicked() {
								cm.OnMarkRead(row.ID, live.ActivitySeq)
							}
						}
						if preview := cm.Activity[row.ID].Preview; preview != "" {
							ui.Text(c, preview).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
						}
						for _, cached := range m.DshbotRoster {
							if cached.ID == row.ID && cached.Detail != "" {
								ui.Text(c, cached.Detail).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
								break
							}
						}
						ui.Row(c).Gap(6).Children(func() {
							if m.OnDshbotSelect != nil && ui.Button(c, i18n.T(i18n.KeyW4AssistantSelect)).Disabled(blocked || row.ID == m.DshbotSelectedID).Clicked() {
								m.OnDshbotSelect(row.ID)
							}
							if m.OnDshbotOpenRosterSession != nil && ui.Button(c, i18n.T(i18n.KeyW4AssistantOpenConversation)).Disabled(blocked).Clicked() {
								m.OnDshbotOpenRosterSession(row.ID)
							}
							label := i18n.T(i18n.KeyW4AssistantPin)
							if row.Pinned {
								label = i18n.T(i18n.KeyW4AssistantUnpin)
							}
							if ui.Button(c, label).Disabled(readonly).Clicked() {
								action(pluginrpc.DshbotContactCommand{Action: "pin", ID: row.ID, SessionID: row.SessionID, Pinned: !row.Pinned})
							}
							if ui.Button(c, i18n.T(i18n.KeyW4AssistantMoveToSection)).Disabled(readonly).Clicked() {
								change("assign", row.ID)
							}
						})
						if s.AssignmentItem == row.ID {
							ui.Column(c).Gap(4).Children(func() {
								if ui.Button(c, i18n.T(i18n.KeyW4AssistantMoveToUnassigned)).Disabled(readonly || pluginrpc.DshbotContactText(row, "sectionId") == "").Clicked() {
									action(pluginrpc.DshbotContactCommand{Action: "assign", ID: row.ID, SessionID: row.SessionID})
								}
								for _, section := range s.Catalog.Sections {
									section := section
									if ui.Button(c, section.Name).Disabled(readonly || pluginrpc.DshbotContactText(row, "sectionId") == section.ID).Clicked() {
										action(pluginrpc.DshbotContactCommand{Action: "assign", ID: row.ID, SessionID: row.SessionID, SectionID: section.ID})
									}
								}
								if ui.Button(c, i18n.T(i18n.KeyW4AssistantCancelMove)).Disabled(blocked).Clicked() {
									change("assign", "")
								}
							})
						}
					})
				}
			})
		}
		if count == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoMatchingContacts)).TextColor(muted)
		}
	})
}
