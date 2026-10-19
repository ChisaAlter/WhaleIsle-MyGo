package composer

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	"whaleisle/internal/store/session"
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
func paintGroupedModelCatalog(c *ui.Context, m Model, muted ui.Color, height float32) {
	root := ui.Column(c.Key("grouped-model-catalog"))
	state := ui.Local(root, "menu", func() modelCatalogMenu { return modelCatalogMenu{} })
	triggerFocus := ui.Local(root, "trigger-focus", func() ui.Handle { return ui.Handle{} })
	binding := fmt.Sprintf("%d:%s", m.Snapshot.ComposerHostGeneration, m.Snapshot.SelectedID)
	if state.Binding != binding {
		*state = modelCatalogMenu{Binding: binding}
	}
	root.Children(func() {
		caption := m.Snapshot.SelectedModel
		if caption == "" && m.ModelPick != nil {
			caption = strings.TrimSpace(*m.ModelPick)
		}
		if caption == "" {
			caption = "模型 / Model"
		}
		hasCurrentChoice := false
		for _, label := range m.Snapshot.ModelLabels {
			if label == caption {
				hasCurrentChoice = true
				break
			}
		}
		if m.Snapshot.SelectedModelEffort != "" {
			caption += " · " + m.Snapshot.SelectedModelEffort
		}
		if m.Snapshot.ModelSelectionBusy {
			caption += " …"
		}
		trigger := ui.Button(c, "").Bind(triggerFocus).Label("Model catalog").Tooltip(caption).MaxWidth(220).MinWidth(0).Height(height).FontSize(13).TextColor(muted).Disabled(m.Inert)
		trigger.Children(func() { ui.Text(c, caption).Grow(1).MinWidth(0).SingleLine().FixedLineHeight(20) })

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
		ui.Popover(c, trigger, &state.Open, func() {
			ui.Column(c.Key("model-pane:" + state.Pane)).Width(300).Gap(5).HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputKeyDown && !state.Composing && (ev.Key == ui.KeyEscape && ev.Mods == 0 || ev.Key == ui.KeyTab && ev.Mods == ui.Shift) {
					if state.Pane != "root" && hasCurrentChoice {
						state.Pane, state.Query = "root", ""
						return true
					}
					state.Open = false
					triggerFocus.RequestFocus(c)
					return true
				}
				return false
			}).Children(func() {
				if state.Pane == "root" || state.Pane == "" {
					modelButton := ui.Button(c, composerText("模型", "Model")+"  "+m.Snapshot.SelectedModel+"  ›").FillWidth().Disabled(m.Inert)
					if state.RootFocus != "effort" {
						modelButton.AutoFocus()
					}
					buttons := []ui.Element{modelButton}
					if modelButton.Clicked() || modelButton.Shortcut(0, ui.KeyRight) {
						state.Pane, state.RootFocus, state.Query, state.Active = "model", "model", "", -1
					}
					if len(m.Snapshot.ModelEffortLabels) > 0 {
						effortButton := ui.Button(c, composerText("思考强度", "Effort")+"  "+m.Snapshot.SelectedModelEffort+"  ›").FillWidth().Disabled(m.Inert)
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
					if ui.Button(c, composerText("‹ 返回", "‹ Back")).Clicked() {
						state.Pane, state.Query = "root", ""
					}
				}
				if state.Pane == "effort" {
					var buttons []ui.Element
					for index, effort := range m.Snapshot.ModelEffortLabels {
						label := effort
						if effort == m.Snapshot.SelectedModelEffort {
							label = "✓ " + label
						}
						button := ui.Button(c, label).Label("Choose effort " + effort).FillWidth().Disabled(m.Snapshot.ModelSelectionBusy)
						buttons = append(buttons, button)
						if effort == m.Snapshot.SelectedModelEffort || m.Snapshot.SelectedModelEffort == "" && index == 0 {
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
					ui.TextInput(c, &state.Query).Label("Search models").Placeholder("搜索模型 / Search models").FillWidth().AutoFocus().OnChange(func() { state.Active = 0 }).HandleInput(func(ev ui.InputEvent) bool {
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
							}
							return true
						case ui.KeyUp:
							if !m.Snapshot.ModelSelectionBusy && len(visible) > 0 {
								state.Active = (state.Active + len(visible) - 1) % len(visible)
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
					ui.Column(c).FillWidth().Gap(4).Children(func() {
						index := 0
						var buttons []ui.Element
						for _, g := range groups {
							ui.Text(c, g.Name).FontSize(11).Bold().TextColor(muted)
							for _, choice := range g.Choices {
								label := choice.Name
								if choice.Label == m.Snapshot.SelectedModel {
									label = "✓ " + label
								}
								button := ui.Button(c, label).Label("Choose model " + choice.Label).FillWidth().Disabled(m.Snapshot.ModelSelectionBusy)
								buttons = append(buttons, button)
								if len(m.Snapshot.ModelLabels) <= 4 && (choice.Label == m.Snapshot.SelectedModel || m.Snapshot.SelectedModel == "" && index == 0) {
									button.AutoFocus()
								}
								if len(m.Snapshot.ModelLabels) > 4 && index == state.Active {
									button.BorderWidth(1)
								}
								if button.Clicked() {
									choose(choice.Label)
								}
								index++
							}
						}
						if len(m.Snapshot.ModelLabels) <= 4 {
							modelMenuKeyboard(buttons)
						}
						if len(visible) == 0 {
							ui.Text(c, "没有匹配模型 / No matching models").FontSize(12)
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
			return
		}
	}
}
