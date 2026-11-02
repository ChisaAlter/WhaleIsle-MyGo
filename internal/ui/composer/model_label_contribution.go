package composer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
)

// ModelLabelPlacement is the fixed model-selection slot's display location.
type ModelLabelPlacement string

const (
	ModelLabelTrigger ModelLabelPlacement = "trigger"
	ModelLabelMenu    ModelLabelPlacement = "menu"
	ModelLabelOption  ModelLabelPlacement = "option"
)

func modelSelectBinding(s session.ViewState) string {
	return fmt.Sprintf("%d:%d:%s:%d:%d:%s", s.ComposerHostGeneration, s.ComposerSessionGeneration,
		s.SelectedID, s.NativeContributionsGeneration, s.NativeCommandGeneration("model"), s.NativeContributions.CommandRevision("model"))
}

func catalogModelLabel(s session.ViewState, label string) (session.ModelCatalogChoice, bool) {
	if route, ok := s.ModelChoices[label]; ok {
		for _, choice := range s.ModelCatalogChoices {
			if choice.Label == label && choice.Provider == route.Provider && choice.Name != "" {
				return choice, true
			}
		}
		return session.ModelCatalogChoice{}, false
	}
	for _, choice := range s.ModelCatalogChoices {
		if choice.Name == "" {
			continue
		}
		if choice.Provider != "" && choice.Provider+"/"+choice.Label == label {
			return choice, true
		}
		if choice.Label == label && (strings.Contains(label, "/") || len(s.ModelLabels) == 0) {
			return choice, true
		}
	}
	if len(s.ModelChoices) > 0 {
		keys := make([]string, 0, len(s.ModelChoices))
		seen := make(map[string]bool, len(s.ModelChoices))
		for _, k := range s.ModelLabels {
			if _, ok := s.ModelChoices[k]; ok && !seen[k] {
				keys = append(keys, k)
				seen[k] = true
			}
		}
		if len(keys) < len(s.ModelChoices) {
			extra := make([]string, 0, len(s.ModelChoices)-len(keys))
			for k := range s.ModelChoices {
				if !seen[k] {
					extra = append(extra, k)
				}
			}
			sort.Strings(extra)
			keys = append(keys, extra...)
		}
		resolveKey := func(k string) (session.ModelCatalogChoice, bool) {
			r := s.ModelChoices[k]
			for _, choice := range s.ModelCatalogChoices {
				if choice.Label == k && (choice.Provider == "" || choice.Provider == r.Provider) && choice.Name != "" {
					return choice, true
				}
			}
			if _, name, found := strings.Cut(k, " — "); found && strings.TrimSpace(name) != "" {
				return session.ModelCatalogChoice{Label: k, Name: strings.TrimSpace(name), Provider: r.Provider}, true
			}
			if k != "" && !strings.Contains(k, "/") {
				return session.ModelCatalogChoice{Label: k, Name: k, Provider: r.Provider}, true
			}
			return session.ModelCatalogChoice{}, false
		}
		for _, k := range keys {
			r := s.ModelChoices[k]
			if r.Provider != "" && r.Model != "" && r.Provider+"/"+r.Model == label {
				if choice, ok := resolveKey(k); ok {
					return choice, true
				}
			}
		}
		for _, k := range keys {
			r := s.ModelChoices[k]
			if r.Model != "" && r.Model == label {
				if choice, ok := resolveKey(k); ok {
					return choice, true
				}
			}
		}
	}
	return session.ModelCatalogChoice{}, false
}

func modelLabelName(s session.ViewState, label string) string {
	if choice, ok := catalogModelLabel(s, label); ok {
		return choice.Name
	}
	return label
}

func friendlyModelTriggerName(s session.ViewState, label string) string {
	if choice, ok := catalogModelLabel(s, label); ok && choice.Name != "" {
		return choice.Name
	}
	if _, name, found := strings.Cut(label, " — "); found && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if known := knownModelSlugDisplayName(label); known != "" {
		return known
	}
	if strings.Contains(label, "/") && !strings.Contains(label, " / ") {
		if label == "deepseek-official/deepseek-flash" && len(s.ModelChoices) == 0 && len(s.ModelCatalogChoices) == 0 {
			return label
		}
		if formatted := formatRawModelSlug(label); formatted != "" {
			return formatted
		}
	}
	return label
}

func knownModelSlugDisplayName(slug string) string {
	switch strings.ToLower(strings.TrimSpace(slug)) {
	case "swe-2":
		return "SWE-2"
	case "swe-1.5":
		return "SWE-1.5"
	case "swe-1":
		return "SWE-1"
	case "gpt-6.1-sol":
		return "GPT-6.1-Sol"
	case "gpt-6.1-terra":
		return "GPT-6.1 Terra"
	case "gpt-6-astra":
		return "GPT-6-Astra"
	case "deepseek-v4.1-flash-1017":
		return "DeepSeek V4.1 Flash 1017"
	}
	return ""
}

func formatRawModelSlug(label string) string {
	slug := strings.TrimSpace(label)
	if i := strings.LastIndex(slug, "/"); i >= 0 && i+1 < len(slug) {
		slug = strings.TrimSpace(slug[i+1:])
	}
	if slug == "" {
		return label
	}
	if known := knownModelSlugDisplayName(slug); known != "" {
		return known
	}
	lower := strings.ToLower(slug)
	switch lower {
	case "deepseek-v4.1-flash":
		return "DeepSeek-V4.1-Flash"
	case "deepseek-v4.1-pro":
		return "DeepSeek-V4.1-Pro"
	case "deepseek-chat":
		return "DeepSeek-Chat"
	case "deepseek-reasoner":
		return "DeepSeek-Reasoner"
	}
	if strings.HasPrefix(lower, "swe-") {
		return "SWE-" + slug[4:]
	}
	if strings.HasPrefix(lower, "gpt-") {
		parts := strings.Split(slug, "-")
		for i, part := range parts {
			if i == 0 {
				parts[i] = "GPT"
			} else if len(part) > 0 && part[0] >= 'a' && part[0] <= 'z' {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		return strings.Join(parts, "-")
	}
	return slug
}

func formatEffortDisplay(effort string) string {
	trimmed := strings.TrimSpace(effort)
	if trimmed == "" {
		return ""
	}
	switch strings.ToLower(trimmed) {
	case "max":
		return "Max"
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	case "xhigh", "x-high":
		return "XHigh"
	case "minimal":
		return "Minimal"
	}
	if trimmed[0] >= 'a' && trimmed[0] <= 'z' {
		return strings.ToUpper(trimmed[:1]) + trimmed[1:]
	}
	return trimmed
}

func modelLabel1017(s session.ViewState, label string) bool {
	if !s.NativeContributions.Known || !s.NativeContributions.ModelLabel1017 {
		return false
	}
	choice, ok := catalogModelLabel(s, label)
	if !ok {
		return false
	}
	route := s.ModelChoices[label]
	return (route.Provider == "deepseek-account" || route.Provider == "deepseek-official") &&
		route.Model == "deepseek-flash" && choice.Name == "DeepSeek-V4.1-Flash"
}

// ModelLabel renders the compiled, fixed-source model label contribution. The
// opaque catalog label remains the selection key; only real catalog metadata
// supplies a display name or release badge. Graph admission is owned by the
// native contribution registry, rather than a claim that React has mounted.
// selected is deliberately not a badge condition: matching unselected options
// carry the same marking as the trigger and menu value in the original slot.
func ModelLabel(c *ui.Context, s session.ViewState, label string, placement ModelLabelPlacement, selected bool, color ui.Color, fontSize float32) ui.Element {
	// A slot registration, rather than the selected route, owns the animated
	// name. Badge matching never replaces that owner: the 1017 renderer also
	// renders ordinary names for models that do not carry its release tag.
	decorated := s.NativeContributions.Known && s.NativeContributions.ModelLabel1017
	key := fmt.Sprintf("model-label:%s:%s:%t:%d:%s", placement, modelSelectBinding(s),
		decorated, s.NativeModelLabelGeneration, s.NativeContributions.ModelLabelRevision())
	row := ui.Row(c.Key(key)).
		MinWidth(0).MaxWidthPercent(100).AlignItems(ui.Center).Gap(4)
	row.Children(func() {
		name := modelLabelName(s, label)
		if placement == ModelLabelTrigger {
			name = friendlyModelTriggerName(s, label)
			modelFlipText(c, "trigger-name", name, color, fontSize)
		} else {
			ui.Text(c, name).MinWidth(0).SingleLine().
				FontSize(fontSize).FontWeight(400).TextColor(color)
		}
		if modelLabel1017(s, label) {
			info := c.Theme().Accent
			ui.Text(c, "1017").Shrink(0).SingleLine().FontSize(11).FontWeight(500).
				FixedLineHeight(17).Padding(1, 8).Radius(999).
				TextColor(info).Background(info.Alpha(.1))
		}
	})
	return row
}
