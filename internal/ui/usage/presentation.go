package usage

import (
	"math"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
	"whaleisle/internal/ui/theme"
)

// usageHeatColor derives the 5-level heatmap palette from the active coss
// card and primary tokens.
func usageHeatColor(t *design.Tokens, level int) ui.Color {
	if level <= 0 {
		return t.Card.Mix(t.MutedForeground, 0.14)
	}
	weight := []float32{0, 0.26, 0.48, 0.74, 1}[min(level, 4)]
	return t.Card.Mix(t.Primary, weight)
}

// usageSeriesColors derives the 6-series chart palette from coss semantic chart
// tokens plus muted-foreground for the "Other" bucket.
func usageSeriesColors(t *design.Tokens) []ui.Color {
	return []ui.Color{t.Chart1, t.Chart2, t.Chart3, t.Chart4, t.Chart5, t.MutedForeground}
}

func usageCard(c *ui.Context, key, title string, content func()) ui.Element {
	return coss.Card(c, coss.CardOptions{Key: key, Title: title}, content)
}

func usageEmpty(c *ui.Context, title, hint string) {
	coss.Card(c, coss.CardOptions{}, func() {
		coss.Empty(c, coss.EmptyOptions{
			Icon:        coss.IconChartBar,
			Title:       title,
			Description: hint,
		}, nil)
	})
}

// usageMemoVersion starts a memo version seeded with locale, design tokens,
// theme preferences and store identity.
func usageMemoVersion(m Model) pagekit.Version {
	v := pagekit.NewVersion().Uint(design.Revision()).Uint(theme.Revision())
	if m.Store != nil {
		v = v.String(m.Store.StatisticsOwner())
	}
	return v
}

func mixFloat64(v pagekit.Version, f float64) pagekit.Version {
	return v.Uint(math.Float64bits(f))
}
