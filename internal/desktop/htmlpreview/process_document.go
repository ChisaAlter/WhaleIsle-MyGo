package htmlpreview

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// ProcessDocumentShell only constructs markup. It is not a readiness check or
// permission to execute it. Its sole production caller is the dedicated
// previewprocess helper, after its AppContainer/job handshake. It must never
// be used in the in-process LocalHTMLURL path.
func ProcessDocumentShell(preparedHTML string, width int, appearance string) (string, error) {
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
	return buildShell(request{HTML: preparedHTML, Width: width, Appearance: appearance}, palette), nil
}
