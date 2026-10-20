package market

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// MarketDetails uses the settled native Markdown document, rather than code or
// a browser surface. Original details supplies neither pathImages nor a base
// URL: relative links stay inert and relative images retain their authored alt.
func marketReadme(c *ui.Context, m Model, source string, fg, muted, codeBG, accent ui.Color) {
	if strings.TrimSpace(source) == "" {
		return
	}
	ui.Column(c.Key("market-readme")).Label("Repository README").FillWidth().MinWidth(0).Gap(8).Children(func() {
		// The shell loads MarkdownText CSS statically; Market's plugin factory
		// appends MarketSection CSS later. Its equally specific .readme h3
		// fixes every descendant h3 at14/22 while preserving Markdown's weight
		// and margins. Other Markdown typography follows the content preference.
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketRepositoryReadme), "Repository README")).FontSize(14).FixedLineHeight(22).Bold().TextColor(fg)
		markdown.View(c, source, markdown.Style{
			FontSize: theme.CurrentPreferences().Chat.Size,
			FG:       fg, Muted: muted, CodeBG: codeBG, Accent: accent,
			OnOpenURL: m.OnOpenURL, LiteralHTML: true,
			RenderImage:          func(c *ui.Context, src, alt string) bool { return renderReadmeImage(c, src, alt, m.SVGImages) },
			DocumentLayout:       true,
			DocumentH3FontSize:   14,
			DocumentH3LineHeight: 22,
			QuoteBorder:          chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Caption),
			RuleBorder:           chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Border),
		})
	})
}
