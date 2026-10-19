package market

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
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
		// MarketSection.module.css overrides this section heading to14/22;
		// Markdown body keeps the original shared content-font preference.
		ui.Text(c, marketText("仓库 README", "Repository README")).FontSize(14).FixedLineHeight(22).Bold().TextColor(fg)
		markdown.View(c, source, markdown.Style{
			FontSize: theme.CurrentPreferences().Chat.Size,
			FG:       fg, Muted: muted, CodeBG: codeBG, Accent: accent,
			OnOpenURL: m.OnOpenURL, LiteralHTML: true,
			RenderImage:    renderReadmeImage,
			DocumentLayout: true,
			QuoteBorder:    chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Caption),
			RuleBorder:     chrome.Color(theme.Resolve(m.ThemeID).Active(m.SystemDark).Border),
		})
	})
}
