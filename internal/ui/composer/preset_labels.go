package composer

import (
	"strings"

	"whaleisle/internal/ui/i18n"
)

// builtinPresetName localizes one of the built-in agent preset ids (the Host
// reports the id in AgentPresetLabels and in preset options).
func builtinPresetName(id string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "standard":
		return i18n.T(i18n.KeyComposerPresetStandard), true
	case "ptc":
		return i18n.T(i18n.KeyComposerPresetPTC), true
	case "minimal":
		return i18n.T(i18n.KeyComposerPresetMinimal), true
	case "cordis":
		return i18n.T(i18n.KeyComposerPresetCordis), true
	}
	return "", false
}

// PresetDisplayName localizes the built-in agent preset ids; a name it does not
// know passes through unchanged.
func PresetDisplayName(label string) string {
	if localized, ok := builtinPresetName(label); ok {
		return localized
	}
	return strings.TrimSpace(label)
}

// presetDisplayName is the package-local alias used by the composer.
func presetDisplayName(label string) string { return PresetDisplayName(label) }
