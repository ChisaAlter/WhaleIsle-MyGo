package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	fonttheme "whaleisle/internal/ui/theme"
)

type ContactsModel struct {
	State     dshbotstore.ContactSnapshot
	Activity  map[string]dshbotstore.ContactActivity
	OnRefresh func()
	OnChange  func(field, value string)
	OnAction  func(pluginrpc.DshbotContactCommand)
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
			ui.Text(c, "联系人目录 / Contacts").Bold().TextColor(fg).Grow(1)
			if ui.Button(c, "刷新 / Refresh").Disabled(blocked || s.Scope == "").Clicked() && cm.OnRefresh != nil {
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
		ui.TextInput(c.Key("contacts-search"), &query).Label("搜索名称、标题和说明 / Search contacts").FillWidth().OnChange(func() { change("query", query) })
		ui.Row(c).Gap(6).Children(func() {
			for _, opt := range []struct{ id, label string }{{"", "全部"}, {"bots", "Bot"}, {"rooms", "群聊"}} {
				opt := opt
				if ui.Button(c, opt.label).Disabled(s.Filters.Kind == opt.id).Clicked() {
					change("kind", opt.id)
				}
			}
			hidden := s.Filters.ShowHidden
			if ui.Checkbox(c, &hidden, "显示隐藏项").Changed() {
				change("hidden", fmt.Sprint(hidden))
			}
			if ui.Button(c, "清除筛选").Clicked() {
				change("reset", "")
			}
		})
		ui.Row(c).Gap(6).Children(func() {
			for _, opt := range []struct{ id, label string }{{"", "全部活动"}, {"active", "活跃"}, {"recent", "最近 7 天"}, {"idle", "闲置"}} {
				opt := opt
				if ui.Button(c, opt.label).Disabled(s.Filters.Activity == opt.id).Clicked() {
					change("activity", opt.id)
				}
			}
		})
		sources := dshbotstore.ContactSources(s.Catalog)
		if len(sources) > 0 {
			ui.Text(c, "来源 / Source").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			ui.Column(c).Gap(4).Children(func() {
				if ui.Button(c, "全部来源 / All sources").Disabled(s.Filters.Source == "").Clicked() {
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
			ui.Text(c, "正在读取当前 Host 的完整联系人目录；也可点击刷新。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			return
		}
		ui.Text(c, fmt.Sprintf("目录 revision %d · %d 位联系人 · %d 个分组", s.Catalog.Revision, len(s.Catalog.Items), len(s.Catalog.Sections))).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if !s.Writable {
			ui.Text(c, "Host 当前只读，仍可搜索与打开会话。").TextColor(muted)
		}
		if s.Pending != nil && !s.Busy {
			ui.Text(c, "有待核实的目录操作。刷新会读回结果，不会重发。").TextColor(accent)
		}
		ui.Column(c.Key("contacts-section-editor")).Background(field).Padding(8).Gap(6).Children(func() {
			label := "新建分组 / New section"
			if s.EditingSection != "" {
				label = "重命名分组 / Rename section"
			}
			name := s.SectionName
			ui.TextInput(c.Key("contacts-section-name"), &name).Label(label + " (1–120)").FillWidth().Disabled(readonly).OnChange(func() { change("name", name) })
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c, "保存分组 / Save section").Disabled(readonly || name == "").Clicked() {
					kind := "create"
					if s.EditingSection != "" {
						kind = "rename"
					}
					action(pluginrpc.DshbotContactCommand{Action: kind, ID: s.EditingSection, Name: name})
				}
				if s.EditingSection != "" && ui.Button(c, "取消重命名").Disabled(blocked).Clicked() {
					change("edit", "")
				}
			})
		})
		if s.Undo != nil {
			ui.Text(c, fmt.Sprintf("已删除分组“%s”；联系人保留并移至未分组。撤销会将仍存在的原成员移回该组。", s.Undo.Section.Name)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			if ui.Button(c, "撤销删除分组 / Restore section").Disabled(readonly).Clicked() {
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
						if ui.Button(c, "重命名").Disabled(readonly).Clicked() {
							change("edit", group.ID)
						}
						if ui.Button(c, "上移").Disabled(readonly || index == 0).Clicked() {
							action(pluginrpc.DshbotContactCommand{Action: "move", ID: group.ID, Delta: -1})
						}
						if ui.Button(c, "下移").Disabled(readonly || index >= len(s.Catalog.Sections)-1).Clicked() {
							action(pluginrpc.DshbotContactCommand{Action: "move", ID: group.ID, Delta: 1})
						}
						if ui.Button(c, "删除分组").Disabled(readonly).Clicked() {
							change("delete", group.ID)
						}
					})
					if s.DeleteSection == group.ID {
						ui.Text(c, "只删除分组，所有成员移至未分组；不删除 Bot、群聊或历史。成功后可撤销。").FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
						ui.Row(c).Gap(6).Children(func() {
							if ui.Button(c, "确认删除分组").Disabled(readonly).Clicked() {
								action(pluginrpc.DshbotContactCommand{Action: "delete", ID: group.ID})
							}
							if ui.Button(c, "取消").Disabled(blocked).Clicked() {
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
							title += " (隐藏)"
						}
						if row.ID == m.DshbotSelectedID {
							title = "• " + title
						}
						kind := "Bot"
						if row.Kind == "room" {
							kind = "群聊"
						}
						ui.Text(c, kind+" · "+title).TextColor(fg).Bold()
						if title != row.Name && row.Kind != "room" {
							ui.Text(c, "@"+row.Name).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						}
						if description := pluginrpc.DshbotContactText(row, "description"); description != "" {
							ui.Text(c, description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted).Selectable()
						}
						if _, label := pluginrpc.DshbotContactSource(row); label != "" {
							ui.Text(c, "来源: "+label).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						}
						if live := cm.Activity[row.ID]; live.Active {
							ui.Text(c, "活跃 / Active").FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
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
							if m.OnDshbotSelect != nil && ui.Button(c, "选择").Disabled(blocked || row.ID == m.DshbotSelectedID).Clicked() {
								m.OnDshbotSelect(row.ID)
							}
							if m.OnDshbotOpenRosterSession != nil && ui.Button(c, "打开会话").Disabled(blocked).Clicked() {
								m.OnDshbotOpenRosterSession(row.ID)
							}
							label := "置顶"
							if row.Pinned {
								label = "取消置顶"
							}
							if ui.Button(c, label).Disabled(readonly).Clicked() {
								action(pluginrpc.DshbotContactCommand{Action: "pin", ID: row.ID, SessionID: row.SessionID, Pinned: !row.Pinned})
							}
							if ui.Button(c, "移动到分组").Disabled(readonly).Clicked() {
								change("assign", row.ID)
							}
						})
						if s.AssignmentItem == row.ID {
							ui.Column(c).Gap(4).Children(func() {
								if ui.Button(c, "移至未分组").Disabled(readonly || pluginrpc.DshbotContactText(row, "sectionId") == "").Clicked() {
									action(pluginrpc.DshbotContactCommand{Action: "assign", ID: row.ID, SessionID: row.SessionID})
								}
								for _, section := range s.Catalog.Sections {
									section := section
									if ui.Button(c, section.Name).Disabled(readonly || pluginrpc.DshbotContactText(row, "sectionId") == section.ID).Clicked() {
										action(pluginrpc.DshbotContactCommand{Action: "assign", ID: row.ID, SessionID: row.SessionID, SectionID: section.ID})
									}
								}
								if ui.Button(c, "取消移动").Disabled(blocked).Clicked() {
									change("assign", "")
								}
							})
						}
					})
				}
			})
		}
		if count == 0 {
			ui.Text(c, "没有符合筛选的联系人 / No matching contacts").TextColor(muted)
		}
	})
}
