package htmlpreview

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

// ProcessDocumentShell only constructs markup. It is not a readiness check or
// permission to execute it. Its sole production caller is the dedicated
// previewprocess helper, after its AppContainer/job handshake. It must never
// be used in the in-process LocalHTMLURL path.
func ProcessDocumentShell(preparedHTML string, width int, appearance string) (string, error) {
	return processDocumentShell(preparedHTML, width, appearance, false)
}

// ProcessImageDocumentShell keeps the same opaque iframe and trusted readiness
// bootstrap, and additionally reports its sole inert image's natural size.
// Its caller supplies only the fixed data-URL image document, never SVG markup
// as executable page content. Supplied image bytes remain script-inert.
func ProcessImageDocumentShell(preparedHTML string, width int, appearance string) (string, error) {
	return processDocumentShell(preparedHTML, width, appearance, true)
}

func processDocumentShell(preparedHTML string, width int, appearance string, inertImage bool) (string, error) {
	if len(preparedHTML) > MaxPreparedHTMLBytes || !utf8.ValidString(preparedHTML) || strings.TrimSpace(preparedHTML) == "" {
		return "", errors.New("invalid prepared HTML snapshot")
	}
	if width < MinWidth || width > MaxWidth || (appearance != "light" && appearance != "dark") {
		return "", errors.New("invalid preview dimensions or appearance")
	}
	palette := previewPalette{Background: "#ffffff", Surface: "#f5f5f5", Text: "#171717", Secondary: "#666666", Border: "#dddddd", Accent: "#2563eb"}
	if appearance == "dark" {
		palette = previewPalette{Background: "#171717", Surface: "#242424", Text: "#eeeeee", Secondary: "#aaaaaa", Border: "#444444", Accent: "#60a5fa"}
	}
	return buildShellMode(request{HTML: preparedHTML, Width: width, Appearance: appearance, inertImage: inertImage}, palette, true), nil
}

// ProcessThemeScript changes the existing iframe, never replacing its document.
func ProcessThemeScript(appearance string) (string, error) {
	return ProcessThemeTokensScript(appearance, nil)
}
func ProcessThemeTokensScript(appearance string, variables map[string]string) (string, error) {
	if appearance != "light" && appearance != "dark" {
		return "", errors.New("invalid preview appearance")
	}
	palette := previewPalette{Background: "#ffffff", Surface: "#f5f5f5", Text: "#171717", Secondary: "#666666", Border: "#dddddd", Accent: "#2563eb"}
	if appearance == "dark" {
		palette = previewPalette{Background: "#171717", Surface: "#242424", Text: "#eeeeee", Secondary: "#aaaaaa", Border: "#444444", Accent: "#60a5fa"}
	}
	tokens := map[string]string{"--bg": palette.Background, "--background": palette.Background, "--surface": palette.Surface, "--text": palette.Text, "--text-secondary": palette.Secondary, "--border": palette.Border, "--accent": palette.Accent}
	for key, value := range variables {
		if _, ok := tokens[key]; !ok || len(value) > 128 || value == "" {
			return "", errors.New("invalid preview theme token")
		}
		tokens[key] = value
	}
	payload, _ := json.Marshal(map[string]any{"type": "whale-visual-reply-theme", "appearance": appearance, "variables": tokens})
	return "window.__previewTheme(" + string(payload) + ");true", nil
}
