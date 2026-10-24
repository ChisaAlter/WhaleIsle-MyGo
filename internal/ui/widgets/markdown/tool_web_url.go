package markdown

import (
	"strings"

	"github.com/egoist/mygo/ui"
	browserurl "github.com/nlnwa/whatwg-url/url"
)

// WebHref uses the browser URL parser and retains the recorded spelling. Only
// absolute HTTP(S) destinations become retrieval links; parsing does not navigate.
func WebHref(raw string) string {
	parsed, err := browserurl.Parse(raw)
	if err != nil || parsed.Scheme() != "http" && parsed.Scheme() != "https" {
		return ""
	}
	return raw
}

// WebLinkLabel retains a supplied title, otherwise uses the browser hostname,
// including www, ASCII domain conversion and brackets around IPv6 addresses.
func WebLinkLabel(raw, title string) string {
	if title != "" {
		return title
	}
	if parsed, err := browserurl.Parse(raw); err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return raw
}

// externalURLActivated keeps browser-anchor keyboard semantics in web cards and
// their Markdown answers. Claim Space before an outer disclosure can handle it.
func externalURLActivated(element ui.Element) bool {
	space := element.Shortcut(0, ui.KeySpace)
	enter := element.Shortcut(0, ui.KeyEnter)
	clicked := element.Clicked()
	return !space && (clicked || enter)
}

func webInlineText(raw string) string {
	// CSS white-space:normal collapses only ASCII whitespace. NBSP and all other
	// provider characters remain visible. Leading/trailing collapsible space does
	// not affect the line's painted content.
	return strings.Join(strings.FieldsFunc(raw, func(r rune) bool { return r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '\f' }), " ")
}
