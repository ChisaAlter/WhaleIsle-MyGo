// Package custompanel renders bounded, native declarative user Panels.
package custompanel

import (
	"whaleisle/internal/ui/widgets"
	"fmt"
	"github.com/egoist/mygo/ui"
	panel "whaleisle/internal/store/custompanel"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type Editor struct {
	Requirements string
	JSON         string
	EditingID    string
	Template     bool
	Preview      *panel.Definition
	Error        string
	Notice       string
}
type Model struct {
	ThemeID                                      string
	SystemDark                                   bool
	BotLabel                                     string
	Available                                    bool
	State                                        panel.Snapshot
	Editor                                       *Editor
	OnCreate, OnImport, OnPreview, OnSave, OnNew func()
	OnAdd                                        func(string)
	OnEdit                                       func(panel.Definition, bool)
	OnUpdate                                     func(panel.Instance)
	OnTemplate                                   func(panel.Definition)
	OnMove                                       func(string, int)
	OnRemove                                     func(string)
	OnPrompt                                     func(string)
}

func View(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().Padding(10).Gap(10).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscCustompanelCustomPanels)).Bold().TextColor(fg)
			if m.Editor == nil {
				return
			}
			e := m.Editor
			if !m.Available {
				ui.Text(c, i18n.T(i18n.KeyMiscCustompanelOpenABotSCanonicalConversation)).TextColor(muted)
			} else {
				ui.Text(c, m.BotLabel).TextColor(muted)
			}
			if e.Error != "" {
				ui.Text(c, e.Error).TextColor(chrome.Color(colors.Accent)).Selectable()
			}
			widgets.Notice(c, "panel:60", "", e.Notice != "", func() string { return e.Notice })
			if !m.Available {
				return
			}
			ui.TextArea(c.Key("panel-requirements"), &e.Requirements).Label(i18n.T(i18n.KeyMiscCustompanelRequirements)).Lines(2, 5).FillWidth()
			if ui.Button(c, i18n.T(i18n.KeyMiscCustompanelCreateWithBot)).Disabled(m.OnCreate == nil).Clicked() {
				m.OnCreate()
			}
			ui.Text(c, i18n.T(i18n.KeyMiscCustompanelSendThePreparedDraftThenImportThe)).FontSize(11).TextColor(muted)
			ui.Row(c).Gap(4).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyMiscCustompanelImportReply)).Disabled(m.OnImport == nil).Clicked() {
					m.OnImport()
				}
				if ui.Button(c, i18n.T(i18n.KeyMiscCustompanelNewJson)).Disabled(m.OnNew == nil).Clicked() {
					m.OnNew()
				}
			})
			mode := i18n.T(i18n.KeyMiscCustompanelNewTemplateAndBotCopy)
			if e.EditingID != "" {
				mode = i18n.T(i18n.KeyMiscCustompanelEditThisBotSCopy)
				if e.Template {
					mode = "编辑库模板（已有副本不变）/ Edit template for future copies"
				}
			}
			ui.Text(c, mode).FontSize(11).TextColor(muted)
			ui.TextArea(c.Key("panel-json"), &e.JSON).Label("Panel JSON").Lines(4, 10).FillWidth().OnChange(func() { e.Preview = nil })
			ui.Row(c).Gap(4).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyMiscCustompanelPreview)).Disabled(m.OnPreview == nil).Clicked() {
					m.OnPreview()
				}
				if ui.Button(c, i18n.T(i18n.KeyMiscCustompanelSave)).Disabled(e.Preview == nil || m.OnSave == nil).Clicked() {
					m.OnSave()
				}
			})
			if e.Preview != nil {
				ui.Text(c, "预览（不运行操作）/ Preview").TextColor(muted)
				render(c.Key("preview"), panel.Instance{Definition: *e.Preview, State: map[string]string{}}, nil, nil, fg, border)
			}
			ui.Text(c, i18n.T(i18n.KeyMiscCustompanelThisBotSPanels)).Bold().TextColor(fg)
			for _, item := range m.State.Instances {
				i := item
				pc := c.Key("instance-" + i.Definition.ID)
				ui.Column(pc).FillWidth().Gap(4).Children(func() {
					ui.Row(pc).Gap(4).Children(func() {
						if ui.Button(pc, "↑").Clicked() && m.OnMove != nil {
							m.OnMove(i.Definition.ID, -1)
						}
						if ui.Button(pc, "↓").Clicked() && m.OnMove != nil {
							m.OnMove(i.Definition.ID, 1)
						}
						if ui.Button(pc, i18n.T(i18n.KeyMiscCustompanelEdit)).Clicked() && m.OnEdit != nil {
							m.OnEdit(i.Definition, false)
						}
						if ui.Button(pc, i18n.T(i18n.KeyMiscCustompanelRemove)).Clicked() && m.OnRemove != nil {
							m.OnRemove(i.Definition.ID)
						}
					})
					ui.Row(pc).Gap(4).Children(func() {
						if ui.Button(pc, "宽 −").Disabled(i.Definition.Width <= 240).Clicked() {
							i.Definition.Width = max(240, i.Definition.Width-40)
							m.OnUpdate(i)
						}
						if ui.Button(pc, "宽 +").Disabled(i.Definition.Width >= 1600).Clicked() {
							i.Definition.Width = min(1600, i.Definition.Width+40)
							m.OnUpdate(i)
						}
						if ui.Button(pc, "高 −").Disabled(i.Definition.Height <= 120).Clicked() {
							i.Definition.Height = max(120, i.Definition.Height-40)
							m.OnUpdate(i)
						}
						if ui.Button(pc, "高 +").Disabled(i.Definition.Height >= 1600).Clicked() {
							i.Definition.Height = min(1600, i.Definition.Height+40)
							m.OnUpdate(i)
						}
					})
					if ui.Button(pc, i18n.T(i18n.KeyMiscCustompanelUpdateLibraryTemplate)).Clicked() && m.OnTemplate != nil {
						m.OnTemplate(i.Definition)
					}
					render(pc, i, m.OnUpdate, m.OnPrompt, fg, border)
				})
			}
			ui.Text(c, i18n.T(i18n.KeyMiscCustompanelPanelLibrary)).Bold().TextColor(fg)
			for _, definition := range m.State.Library {
				d := definition
				lc := c.Key("library-" + d.ID)
				ui.Column(lc).Gap(4).Children(func() {
					ui.Text(lc, d.Title).TextColor(fg)
					ui.Row(lc).Gap(4).Children(func() {
						added := false
						for _, i := range m.State.Instances {
							if i.Definition.ID == d.ID {
								added = true
							}
						}
						if ui.Button(lc, i18n.T(i18n.KeyMiscCustompanelAddCopy)).Disabled(added || m.OnAdd == nil).Clicked() {
							m.OnAdd(d.ID)
						}
						if ui.Button(lc, i18n.T(i18n.KeyMiscCustompanelEditTemplate)).Disabled(m.OnEdit == nil).Clicked() {
							m.OnEdit(d, true)
						}
					})
				})
			}
		})
	})
}
func render(c *ui.Context, i panel.Instance, onUpdate func(panel.Instance), onPrompt func(string), fg, border ui.Color) {
	d := i.Definition
	ui.Column(c).Width(float32(d.Width)).MaxWidthPercent(100).Height(float32(d.Height)).MinHeight(120).Border(1, border).Radius(6).Padding(8).Gap(6).Children(func() {
		ui.Text(c, fmt.Sprintf("%s · %d×%d", d.Title, d.Width, d.Height)).Bold().TextColor(fg)
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				for start := 0; start < len(d.Widgets); start += d.Columns {
					ui.Row(c).FillWidth().Gap(8).Children(func() {
						for n := start; n < min(start+d.Columns, len(d.Widgets)); n++ {
							w := d.Widgets[n]
							wc := c.Key(w.ID)
							ui.Column(wc).Grow(1).MinWidth(0).Gap(4).Children(func() {
								value, exists := i.State[w.ID]
								if !exists {
									value = w.Text
								}
								change := func(v string) {
									if onUpdate == nil {
										return
									}
									if i.State == nil {
										i.State = map[string]string{}
									}
									i.State[w.ID] = v
									onUpdate(i)
								}
								size := w.FontSize
								if size == 0 {
									size = 13
								}
								switch w.Kind {
								case "text":
									ui.Text(wc, w.Text).FontSize(float32(size)).TextColor(fg).Selectable()
								case "input":
									ui.TextArea(wc, &value).Label(w.Label).Lines(2, 6).FillWidth().Disabled(onUpdate == nil).OnChange(func() { change(value) })
								case "checkbox":
									checked := value == "true"
									if ui.Checkbox(wc, &checked, w.Label).Disabled(onUpdate == nil).Changed() {
										change(fmt.Sprint(checked))
									}
								case "progress":
									ui.Text(wc, w.Label).TextColor(fg)
									ui.Progress(wc, w.Value).FillWidth()
								case "select":
									ui.Text(wc, w.Label).TextColor(fg)
									if ui.Select(wc, &value, w.Options).Disabled(onUpdate == nil).Changed() {
										change(value)
									}
								case "action":
									if ui.Button(wc, w.Label).Disabled(onPrompt == nil).Tooltip(i18n.T(i18n.KeyMiscCustompanelPrepareChatDraftDoesNotSend)).Clicked() {
										onPrompt(w.Text)
									}
								}
							})
						}
					})
				}
			})
		})
	})
}
