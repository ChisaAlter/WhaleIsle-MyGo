package chat

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

func toolWebCard(c *ui.Context, m *toolWebModel, st ToolCardStyle, key string, openURL func(string)) {
	data := markdown.WebData{Kind: m.Kind, Answer: m.Answer, URL: m.URL, StatusCode: m.StatusCode, Truncated: m.Truncated, Sources: make([]markdown.WebSource, len(m.Sources))}
	for i, source := range m.Sources {
		data.Sources[i] = markdown.WebSource{URL: source.URL, Title: source.Title, Snippet: source.Snippet, PublishedAt: source.PublishedAt}
	}
	background, secondary := st.Field, st.Muted
	if st.Colors.CodeBlock != "" {
		background = chrome.Color(st.Colors.CodeBlock)
		secondary = chrome.Color(st.Colors.Secondary)
	}
	link := ui.Hex("#4176e6")
	if darkSurface(st.FG) {
		link = ui.Hex("#7aaaff")
	}
	style := transcriptMarkdownStyle(st.FG, secondary, background, link, theme.RenderPreferences().Chat.Size, darkSurface(st.FG))
	style.ToolCodeLanguage = st.Muted
	style.OnOpenURL = openURL
	labels := markdown.WebLabels{NoResults: i18n.T(i18n.KeyToolWebEmpty), SourcesTruncated: i18n.T(i18n.KeyToolWebSourcesTruncated), HTTP: i18n.T(i18n.KeyToolWebHTTP), ContentTruncated: i18n.T(i18n.KeyToolWebContentTruncated)}
	markdown.ToolWebBlock(c.Key("tool-web:"+key), data, style, labels).Margin(4, 0, 4, 4).Label(i18n.T(i18n.KeyToolWebCard))
}

// The header links to the producing fetch argument, independently of a redirected
// final URL in metadata. Result-only history has no retained call argument link.
func toolWebFetchHref(call *session.ToolCallView) string {
	if call == nil || call.Name != "web_fetch" || call.Call == nil {
		return ""
	}
	args, ok := terminalModelObject([]byte(call.Arguments))
	if !ok {
		return ""
	}
	raw, ok := terminalModelString(args["url"])
	if !ok {
		return ""
	}
	return markdown.WebHref(raw)
}
