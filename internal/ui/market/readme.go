package market

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// MarketDetails uses the settled native Markdown document, rather than code or
// a browser surface. Original details supplies neither pathImages nor a base
// URL: relative links stay inert and relative images retain their authored alt.
func marketReadme(c *ui.Context, m Model, source string) {
	if strings.TrimSpace(source) == "" {
		return
	}
	t := design.For(c)
	ui.Column(c.Key("market-readme")).Label("Repository README").FillWidth().MinWidth(0).Gap(design.Sp(2)).Children(func() {
		// The shell loads MarkdownText CSS statically; Market's plugin factory
		// appends MarketSection CSS later. Its equally specific .readme h3
		// fixes every descendant h3 at14/22 while preserving Markdown's weight
		// and margins. Other Markdown typography follows the content preference.
		ui.Text(c, marketText(i18n.T(i18n.KeyW4MarketRepositoryReadme), "Repository README")).FontSize(design.TextSM.Size).FixedLineHeight(22).FontWeight(design.WeightBold).TextColor(t.Foreground)
		markdown.View(c, source, markdown.Style{
			FontSize:             theme.CurrentPreferences().Chat.Size,
			FG:                   t.Foreground,
			Muted:                t.MutedForeground,
			CodeBG:               t.Code,
			Accent:               t.Primary,
			OnOpenURL:            m.OnOpenURL,
			LiteralHTML:          true,
			RenderImage:          func(c *ui.Context, src, alt string) bool { return renderReadmeImage(c, src, alt, m.Media) },
			DocumentLayout:       true,
			DocumentH3FontSize:   design.TextSM.Size,
			DocumentH3LineHeight: 22,
			QuoteBorder:          t.Border,
			RuleBorder:           t.Border,
		})
	})
}
