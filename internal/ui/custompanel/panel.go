// Package custompanel renders bounded, native declarative user Panels.
package custompanel

import (
	"fmt"
	"maps"

	"github.com/egoist/mygo/ui"
	panel "whaleisle/internal/store/custompanel"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type Editor struct {
	Requirements string
	JSON         string
	EditingID    string
	Template     bool
	Preview      *panel.Definition
	Failure      Failure
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
			if e.Failure.Sentence != "" {
				widgets.Alert(c, widgets.AlertOptions{ID: "panel:58", Message: e.Failure.Sentence, Detail: e.Failure.Detail})
			}
			widgets.AckLine(c, "panel:60", e.Notice)
			if !m.Available {
				return
			}
			coss.Textarea(c.Key("panel-requirements"), &e.Requirements, coss.TextareaOptions{Label: i18n.T(i18n.KeyMiscCustompanelRequirements), MaxLines: 5})
			if coss.Button(c, i18n.T(i18n.KeyMiscCustompanelCreateWithBot), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Disabled: m.OnCreate == nil}).Clicked() {
				m.OnCreate()
			}
			ui.Text(c, i18n.T(i18n.KeyMiscCustompanelSendThePreparedDraftThenImportThe)).FontSize(11).TextColor(muted)
			ui.Row(c).Gap(4).Children(func() {
				if coss.Button(c, i18n.T(i18n.KeyMiscCustompanelImportReply), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: m.OnImport == nil}).Clicked() {
					m.OnImport()
				}
				if coss.Button(c, i18n.T(i18n.KeyMiscCustompanelNewJson), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: m.OnNew == nil}).Clicked() {
					m.OnNew()
				}
			})
			mode := i18n.T(i18n.KeyMiscCustompanelNewTemplateAndBotCopy)
			if e.EditingID != "" {
				mode = i18n.T(i18n.KeyMiscCustompanelEditThisBotSCopy)
				if e.Template {
					mode = i18n.T(i18n.KeyMiscCustompanelEditLibraryTemplate)
				}
			}
			ui.Text(c, mode).FontSize(11).TextColor(muted)
			coss.Textarea(c.Key("panel-json"), &e.JSON, coss.TextareaOptions{Label: "Panel JSON", MaxLines: 10}).OnChange(func() { e.Preview = nil })
			ui.Row(c).Gap(4).Children(func() {
				if coss.Button(c, i18n.T(i18n.KeyMiscCustompanelPreview), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: m.OnPreview == nil}).Clicked() {
					m.OnPreview()
				}
				if coss.Button(c, i18n.T(i18n.KeyMiscCustompanelSave), coss.ButtonOptions{Variant: coss.ButtonDefault, Size: coss.ButtonSM, Disabled: e.Preview == nil || m.OnSave == nil}).Clicked() {
					m.OnSave()
				}
			})
			if e.Preview != nil {
				ui.Text(c, i18n.T(i18n.KeyMiscCustompanelPreviewOnly)).TextColor(muted)
				render(c.Key("preview"), panel.Instance{Definition: *e.Preview, State: map[string]string{}}, nil, nil, fg, border)
			}
			ui.Text(c, i18n.T(i18n.KeyMiscCustompanelThisBotSPanels)).Bold().TextColor(fg)
			for _, item := range m.State.Instances {
				i := item
				pc := c.Key("instance-" + i.Definition.ID)
				ui.Column(pc).FillWidth().Gap(4).Children(func() {
					ui.Row(pc).Gap(4).Children(func() {
						if coss.Button(pc, "↑", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() && m.OnMove != nil {
							m.OnMove(i.Definition.ID, -1)
						}
						if coss.Button(pc, "↓", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() && m.OnMove != nil {
							m.OnMove(i.Definition.ID, 1)
						}
						if coss.Button(pc, i18n.T(i18n.KeyMiscCustompanelEdit), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() && m.OnEdit != nil {
							m.OnEdit(i.Definition, false)
						}
						if coss.Button(pc, i18n.T(i18n.KeyMiscCustompanelRemove), coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonXS}).Clicked() && m.OnRemove != nil {
							m.OnRemove(i.Definition.ID)
						}
					})
					ui.Row(pc).Gap(4).Children(func() {
						if coss.Button(pc, i18n.Pick("宽 −", "Width −"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Disabled: m.OnUpdate == nil || i.Definition.Width <= 240}).Clicked() && m.OnUpdate != nil {
							i.Definition.Width = max(240, i.Definition.Width-40)
							m.OnUpdate(i)
						}
						if coss.Button(pc, i18n.Pick("宽 +", "Width +"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Disabled: m.OnUpdate == nil || i.Definition.Width >= 1600}).Clicked() && m.OnUpdate != nil {
							i.Definition.Width = min(1600, i.Definition.Width+40)
							m.OnUpdate(i)
						}
						if coss.Button(pc, i18n.Pick("高 −", "Height −"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Disabled: m.OnUpdate == nil || i.Definition.Height <= 120}).Clicked() && m.OnUpdate != nil {
							i.Definition.Height = max(120, i.Definition.Height-40)
							m.OnUpdate(i)
						}
						if coss.Button(pc, i18n.Pick("高 +", "Height +"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS, Disabled: m.OnUpdate == nil || i.Definition.Height >= 1600}).Clicked() && m.OnUpdate != nil {
							i.Definition.Height = min(1600, i.Definition.Height+40)
							m.OnUpdate(i)
						}
					})
					if coss.Button(pc, i18n.T(i18n.KeyMiscCustompanelUpdateLibraryTemplate), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() && m.OnTemplate != nil {
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
						if coss.Button(lc, i18n.T(i18n.KeyMiscCustompanelAddCopy), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: added || m.OnAdd == nil}).Clicked() {
							m.OnAdd(d.ID)
						}
						if coss.Button(lc, i18n.T(i18n.KeyMiscCustompanelEditTemplate), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: m.OnEdit == nil}).Clicked() {
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
									next := maps.Clone(i.State)
									if next == nil {
										next = map[string]string{}
									}
									next[w.ID] = v
									i.State = next
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
									coss.Textarea(wc, &value, coss.TextareaOptions{Label: w.Label, MaxLines: 6, Disabled: onUpdate == nil}).OnChange(func() { change(value) })
								case "checkbox":
									checked := value == "true"
									if coss.Checkbox(wc, &checked, w.Label, coss.CheckOptions{Disabled: onUpdate == nil}).Changed() {
										change(fmt.Sprint(checked))
									}
								case "progress":
									ui.Text(wc, w.Label).TextColor(fg)
									ui.Progress(wc, w.Value).FillWidth()
								case "select":
									ui.Text(wc, w.Label).TextColor(fg)
									opts := make([]coss.SelectOption, len(w.Options))
									for idx, opt := range w.Options {
										opts[idx] = coss.SelectOption{Value: opt, Label: opt}
									}
									if coss.Select(wc, &value, opts, coss.SelectOptions{Disabled: onUpdate == nil}).Changed() {
										change(value)
									}
								case "action":
									if coss.Button(wc, w.Label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: onPrompt == nil}).Tooltip(i18n.T(i18n.KeyMiscCustompanelPrepareChatDraftDoesNotSend)).Clicked() {
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
