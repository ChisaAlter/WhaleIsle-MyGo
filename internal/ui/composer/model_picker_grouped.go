package composer

import (
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type modelCatalogGroup struct {
	Provider, Name string
	Choices        []session.ModelCatalogChoice
}
type modelCatalogMenu struct {
	Binding, Query, Pane string
	RootFocus            string
	Open                 bool
	Active               int
	HoveredIndex         int
	ScrollActive         bool
	Composing            bool
}

// Match original ordered-subsequence scoring, retaining catalog order on ties.
func modelNameScore(name, query string) (int, bool) {
	n, q := []rune(strings.ToLower(name)), []rune(query)
	if len(q) == 0 {
		return 0, true
	}
	if len(q) > len(n) {
		return 0, false
	}
	const absent = -1 << 30
	bonus := func(i int) int {
		if i == 0 || n[i-1] == '-' || n[i-1] == '_' {
			return 8
		}
		return 0
	}
	prev := make([]int, len(n))
	for i := range n {
		prev[i] = absent
		if n[i] == q[0] {
			prev[i] = 1 + bonus(i) - i
		}
	}
	for j := 1; j < len(q); j++ {
		cur := make([]int, len(n))
		left, leftLeft, gapped := absent, absent, absent
		for i, prior := range prev {
			cur[i] = absent
			if leftLeft != absent {
				gapped = max(gapped, leftLeft+i-2)
			}
			if n[i] == q[j] {
				b := 1 + bonus(i)
				if left != absent {
					cur[i] = left + b + 4
				}
				if gapped != absent {
					cur[i] = max(cur[i], gapped+b+1-i)
				}
			}
			leftLeft, left = left, prior
		}
		prev = cur
	}
	best := absent
	for _, score := range prev {
		best = max(best, score)
	}
	return best, best != absent
}
func groupedModelChoices(s session.ViewState, query string) []modelCatalogGroup {
	metadata := map[string]session.ModelCatalogChoice{}
	for _, choice := range s.ModelCatalogChoices {
		metadata[choice.Label] = choice
	}
	groups := []modelCatalogGroup{}
	positions := map[string]int{}
	for _, label := range s.ModelLabels {
		choice, ok := metadata[label]
		if !ok {
			route := s.ModelChoices[label]
			choice = session.ModelCatalogChoice{Label: label, Name: label, Provider: route.Provider, ProviderName: route.Provider}
			if _, name, found := strings.Cut(label, " — "); found {
				choice.Name = name
			}
		}
		if choice.Name == "" {
			choice.Name = choice.Label
		}
		if choice.ProviderName == "" {
			choice.ProviderName = choice.Provider
		}
		index, ok := positions[choice.Provider]
		if !ok {
			index = len(groups)
			positions[choice.Provider] = index
			groups = append(groups, modelCatalogGroup{Provider: choice.Provider, Name: choice.ProviderName})
		}
		groups[index].Choices = append(groups[index].Choices, choice)
	}
	priority := func(p string) int {
		if p == "deepseek-account" {
			return 0
		}
		if p == "deepseek-official" {
			return 1
		}
		return 2
	}
	sort.SliceStable(groups, func(i, j int) bool { return priority(groups[i].Provider) < priority(groups[j].Provider) })
	if len(s.ModelLabels) <= 4 {
		query = ""
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return groups
	}
	filtered := groups[:0]
	for _, g := range groups {
		type ranked struct {
			choice session.ModelCatalogChoice
			score  int
			prefix bool
		}
		matches := []ranked{}
		for _, choice := range g.Choices {
			if score, ok := modelNameScore(choice.Name, query); ok {
				matches = append(matches, ranked{choice, score, strings.HasPrefix(strings.ToLower(choice.Name), query)})
			}
		}
		sort.SliceStable(matches, func(i, j int) bool {
			if matches[i].prefix != matches[j].prefix {
				return matches[i].prefix
			}
			return matches[i].score > matches[j].score
		})
		g.Choices = nil
		for _, match := range matches {
			g.Choices = append(g.Choices, match.choice)
		}
		if len(g.Choices) > 0 {
			filtered = append(filtered, g)
		}
	}
	return filtered
}
func paintGroupedModelCatalog(c *ui.Context, m Model, colors theme.Colors, height float32) {
	muted := chrome.Color(colors.Muted)
	root := ui.Column(c.Key("grouped-model-catalog"))
	state := ui.Local(root, "menu", func() modelCatalogMenu { return modelCatalogMenu{} })
	triggerFocus := ui.Local(root, "trigger-focus", func() ui.Handle { return ui.Handle{} })
	binding := modelSelectBinding(m.Snapshot)
	if state.Binding != binding {
		*state = modelCatalogMenu{Binding: binding}
	}
	root.Children(func() {
		selectionLabel := m.Snapshot.SelectedModel
		if selectionLabel == "" && m.ModelPick != nil {
			selectionLabel = strings.TrimSpace(*m.ModelPick)
		}
		name := friendlyModelTriggerName(m.Snapshot, selectionLabel)
		if name == "" {
			name = i18n.T(i18n.KeyComposerModelMenuModel)
		}
		caption := name
		hasCurrentChoice := false
		for _, label := range m.Snapshot.ModelLabels {
			if label == selectionLabel {
				hasCurrentChoice = true
				break
			}
		}
		if m.Snapshot.SelectedModelEffort != "" {
			caption += " · " + formatEffortDisplay(m.Snapshot.SelectedModelEffort)
		}
		if m.Snapshot.ModelSelectionBusy {
			caption += " …"
		}
		secondary := muted
		if colors.Secondary != "" {
			secondary = chrome.Color(colors.Secondary)
		}
		trigger := chip(c, colors, state.Open, secondary).Bind(triggerFocus).Label("Model catalog").Tooltip(caption + "  ·  " + i18n.T(i18n.KeyComposerShortcutEsc)).MaxWidth(260).Disabled(m.Inert)
		trigger.Children(func() {
			if hasCurrentChoice {
				ModelLabel(c, m.Snapshot, selectionLabel, ModelLabelTrigger, true, secondary, 13).Grow(1)
			} else {
				modelFlipText(c, "unknown-current:"+binding, name, secondary, 13).Grow(1)
			}
			if m.Snapshot.SelectedModelEffort != "" {
				modelFlipText(c, "trigger-effort:"+binding, formatEffortDisplay(m.Snapshot.SelectedModelEffort), chrome.Color(colors.Caption), 13).Shrink(1000)
			}
			chevron(c, state.Open)
		})

		if trigger.Clicked() {
			state.Open = !state.Open
			state.Query = ""
			state.Active = -1
			state.Pane = "root"
			if !hasCurrentChoice {
				state.Pane = "model"
			}
			state.RootFocus = "model"
		}
		ui.PopoverBase(c, trigger, &state.Open, func(panel ui.Element) {
			panel = menuSurface(c, panel.Padding(4).Gap(0).MinWidth(240).MaxWidth(420).
				AttachTo(trigger, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, 8, 0), colors)
			pane := ui.Column(c.Key("model-pane:" + state.Pane)).MinWidth(232).MaxWidth(412).Gap(0)
			// HandleInput is delivered only to the focused control itself. A
			// subtree shortcut owns Escape from both menu buttons and search,
			// before Popover's fallback dismissal can discard the return focus.
			if !state.Composing && (pane.Shortcut(0, ui.KeyEscape) || pane.Shortcut(ui.Shift, ui.KeyTab)) {
				if state.Pane != "root" && hasCurrentChoice {
					state.Pane, state.Query = "root", ""
				} else {
					state.Open = false
					triggerFocus.RequestFocus(c)
					return
				}
			}
			pane.Children(func() {
				if state.Pane == "root" || state.Pane == "" {
					modelButton := menuRow(c, colors).Label(i18n.T(i18n.KeyComposerModelMenuModel) + "  " + m.Snapshot.SelectedModel + "  ›").Disabled(m.Inert)
					modelButton.Children(func() {
						ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Gap(6).Children(func() {
							ui.Text(c, i18n.T(i18n.KeyComposerModelMenuModel)).FontSize(13).FixedLineHeight(20).TextColor(chrome.Color(colors.FG)).Shrink(0)
							ui.Box(c).Grow(1)
							ModelLabel(c, m.Snapshot, selectionLabel, ModelLabelMenu, true, muted, 13)
							ui.Text(c, "›").FontSize(13).TextColor(muted).Shrink(0)
						})
					})
					if state.RootFocus != "effort" {
						modelButton.AutoFocus()
					}
					buttons := []ui.Element{modelButton}
					if modelButton.Clicked() || modelButton.Shortcut(0, ui.KeyRight) {
						state.Pane, state.RootFocus, state.Query, state.Active = "model", "model", "", -1
					}
					if len(m.Snapshot.ModelEffortLabels) > 0 {
						effortButton := menuRow(c, colors).Label(i18n.T(i18n.KeyComposerModelMenuEffort) + "  " + m.Snapshot.SelectedModelEffort + "  ›").Disabled(m.Inert)
						effortButton.Children(func() {
							ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Gap(6).Children(func() {
								ui.Text(c, i18n.T(i18n.KeyComposerModelMenuEffort)).FontSize(13).FixedLineHeight(20).TextColor(chrome.Color(colors.FG)).Shrink(0)
								ui.Box(c).Grow(1)
								ui.Text(c, formatEffortDisplay(m.Snapshot.SelectedModelEffort)).FontSize(13).FixedLineHeight(20).TextColor(muted).SingleLine()
								ui.Text(c, "›").FontSize(13).TextColor(muted).Shrink(0)
							})
						})
						buttons = append(buttons, effortButton)
						if state.RootFocus == "effort" {
							effortButton.AutoFocus()
						}
						if effortButton.Clicked() || effortButton.Shortcut(0, ui.KeyRight) {
							state.Pane, state.RootFocus = "effort", "effort"
						}
					}
					modelMenuKeyboard(buttons)
					return
				}
				if hasCurrentChoice {
					back := menuRow(c, colors).Label(i18n.T(i18n.KeyComposerModelBack))
					back.Children(func() {
						ui.Text(c, i18n.T(i18n.KeyComposerModelBack)).Grow(1).SingleLine().FontSize(13).FixedLineHeight(20).TextColor(chrome.Color(colors.FG))
					})
					if back.Clicked() {
						state.Pane, state.Query = "root", ""
					}
				}
				if state.Pane == "effort" {
					var buttons []ui.Element
					for index, effort := range m.Snapshot.ModelEffortLabels {
						selected := effort == m.Snapshot.SelectedModelEffort
						button := menuRow(c, colors).Label("Choose effort " + effort).Disabled(m.Snapshot.ModelSelectionBusy)
						button.Children(func() {
							ui.Text(c, effort).Grow(1).SingleLine().FontSize(13).FixedLineHeight(20).TextColor(chrome.Color(colors.FG))
							if selected {
								ui.Icon(c, checkIcon).Size(14, 14).Shrink(0).TextColor(chrome.Color(colors.FG))
							}
						})
						buttons = append(buttons, button)
						if selected || m.Snapshot.SelectedModelEffort == "" && index == 0 {
							button.AutoFocus()
						}
						if button.Clicked() && m.OnModelEffortChange != nil {
							state.Open = false
							triggerFocus.RequestFocus(c)
							m.OnModelEffortChange(effort)
						}
					}
					modelMenuKeyboard(buttons)
					return
				}
				groups := groupedModelChoices(m.Snapshot, state.Query)
				visible := []session.ModelCatalogChoice{}
				for _, g := range groups {
					visible = append(visible, g.Choices...)
				}
				if state.Active < 0 {
					state.Active = 0
					for index, choice := range visible {
						if choice.Label == m.Snapshot.SelectedModel {
							state.Active = index
							break
						}
					}
				}
				if state.Active >= len(visible) {
					state.Active = 0
				}
				choose := func(label string) {
					if m.Inert || m.Snapshot.ModelSelectionBusy || m.OnModelChange == nil {
						return
					}
					state.Open = false
					state.Query = ""
					// Search HandleInput runs between builds, when the captured
					// Element has expired. The stable handle requests the trigger
					// in this window after the closed menu is rebuilt.
					triggerFocus.RequestFocus(c)
					if m.ModelPick != nil {
						*m.ModelPick = label
					}
					m.OnModelChange(label)
				}
				if len(m.Snapshot.ModelLabels) > 4 {
					ui.TextInput(c, &state.Query).Label("Search models").Placeholder(i18n.T(i18n.KeyComposerModelSearch)).FillWidth().Margin(2, 0, 3, 0).Padding(5, 7).Radius(12).BorderWidth(0).Background(ui.Transparent).FontSize(12).AutoFocus().OnChange(func() { state.Active = 0; state.ScrollActive = true }).HandleInput(func(ev ui.InputEvent) bool {
						if ev.Kind == ui.InputCompose {
							state.Composing = ev.Text != ""
							return false
						}
						if ev.Kind != ui.InputKeyDown || state.Composing || ev.Mods != 0 {
							return false
						}
						switch ev.Key {
						case ui.KeyDown:
							if !m.Snapshot.ModelSelectionBusy && len(visible) > 0 {
								state.Active = (state.Active + 1) % len(visible)
								state.ScrollActive = true
							}
							return true
						case ui.KeyUp:
							if !m.Snapshot.ModelSelectionBusy && len(visible) > 0 {
								state.Active = (state.Active + len(visible) - 1) % len(visible)
								state.ScrollActive = true
							}
							return true
						case ui.KeyEnter, ui.KeyTab:
							if len(visible) > 0 {
								choose(visible[state.Active].Label)
							}
							return true
						}
						return false
					})
				}
				ui.Scroll(c).FillWidth().MaxHeight(320).Children(func() {
					ui.Column(c).FillWidth().Gap(0).Children(func() {
						index := 0
						hoveredRow := -1
						var buttons []ui.Element
						for _, g := range groups {
							ui.Text(c, g.Name).FontSize(11).FontWeight(500).FixedLineHeight(16).Padding(5, 8, 2, 8).TextColor(muted)
							for _, choice := range g.Choices {
								selected := choice.Label == m.Snapshot.SelectedModel
								button := menuRow(c, colors).Label("Choose model " + choice.Label).Disabled(m.Snapshot.ModelSelectionBusy)
								if button.Hovered() {
									hoveredRow = index
									if state.HoveredIndex != index {
										state.HoveredIndex = index
										state.Active = index
									}
								}
								if (len(m.Snapshot.ModelLabels) > 4 && index == state.Active) || (len(m.Snapshot.ModelLabels) <= 4 && button.Hovered()) {
									button = button.Background(chrome.Color(colors.Hover))
								}
								if len(m.Snapshot.ModelLabels) > 4 && index == state.Active && state.ScrollActive {
									button.ScrollIntoView()
									state.ScrollActive = false
								}
								button.Children(func() {
									ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
										ModelLabel(c, m.Snapshot, choice.Label, ModelLabelOption, selected, chrome.Color(colors.FG), 13).Grow(1)
										if selected {
											ui.Icon(c, checkIcon).Size(14, 14).Shrink(0).TextColor(chrome.Color(colors.FG))
										}
									})
								})
								buttons = append(buttons, button)
								if len(m.Snapshot.ModelLabels) <= 4 && (choice.Label == m.Snapshot.SelectedModel || m.Snapshot.SelectedModel == "" && index == 0) {
									button.AutoFocus()
								}
								if button.Clicked() {
									choose(choice.Label)
								}
								index++
							}
						}
						if hoveredRow < 0 {
							state.HoveredIndex = -1
						}
						if len(m.Snapshot.ModelLabels) <= 4 {
							modelMenuKeyboard(buttons)
						}
						if len(visible) == 0 {
							ui.Text(c, i18n.T(i18n.KeyComposerModelNoMatch)).FontSize(12).Padding(8).TextColor(muted)
						}
					})
				})
			})
		})
	})
}

func modelMenuKeyboard(buttons []ui.Element) {
	for index, button := range buttons {
		next := -1
		if button.Shortcut(0, ui.KeyDown) {
			next = (index + 1) % len(buttons)
		}
		if button.Shortcut(0, ui.KeyUp) {
			next = (index + len(buttons) - 1) % len(buttons)
		}
		if next >= 0 {
			buttons[next].Focus()
			buttons[next].ScrollIntoView()
			return
		}
	}
}
