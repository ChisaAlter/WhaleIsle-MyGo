package composer

import (
	"fmt"

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
	route, ok := s.ModelChoices[label]
	if !ok {
		return session.ModelCatalogChoice{}, false
	}
	for _, choice := range s.ModelCatalogChoices {
		if choice.Label == label && choice.Provider == route.Provider && choice.Name != "" {
			return choice, true
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
