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
	Binding, Query string
	Open           bool
	Active         int
	Composing      bool
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
		trigger := ui.Button(c, "").Label("Model catalog").Tooltip(caption).MaxWidth(220).MinWidth(0).Height(height).FontSize(13).TextColor(muted)
		trigger.Children(func() { ui.Text(c, caption).Grow(1).MinWidth(0).SingleLine().FixedLineHeight(20) })

		if trigger.Clicked() {
			state.Open = !state.Open
			state.Query = ""
			state.Active = 0
		}
		ui.Popover(c, trigger, &state.Open, func() {
			ui.Column(c).Width(300).Gap(5).Children(func() {
				groups := groupedModelChoices(m.Snapshot, state.Query)
				visible := []session.ModelCatalogChoice{}
				for _, g := range groups {
					visible = append(visible, g.Choices...)
				}
				if state.Active >= len(visible) {
					state.Active = 0
				}
				choose := func(label string) {
					if m.Snapshot.ModelSelectionBusy || m.OnModelChange == nil {
						return
					}
					state.Open = false
					state.Query = ""
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
							if len(visible) > 0 {
								state.Active = (state.Active + 1) % len(visible)
							}
							return true
						case ui.KeyUp:
							if len(visible) > 0 {
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
						for _, g := range groups {
							ui.Text(c, g.Name).FontSize(11).Bold().TextColor(muted)
							for _, choice := range g.Choices {
								label := choice.Name
								if choice.Label == m.Snapshot.SelectedModel {
									label = "✓ " + label
								}
								button := ui.Button(c, label).Label("Choose model " + choice.Label).FillWidth()
								if len(m.Snapshot.ModelLabels) > 4 && index == state.Active {
									button.BorderWidth(1)
								}
								if button.Clicked() {
									choose(choice.Label)
								}
								index++
							}
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
