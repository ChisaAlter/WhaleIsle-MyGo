package market

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const readmeFixture = `# Native project tools

Read **structured documentation** with [Project guide](https://example.invalid/guide) inline.

- Keep the active workspace
- Review ` + "`project.json`" + ` before changing it

| Action | Result |
| --- | --- |
| Inspect | Read only |

` + "```json\n{\"workspace\":\"fixture\",\"description\":\"a deliberately long line that remains inside the native README code scrollport\"}\n```" + `

![Relative illustration](images/local.png)

![Remote illustration](https://example.invalid/screenshot.png)

Raw <b>literal HTML</b> stays text.
`

func TestMarketReadmeNativeDocumentAndExplicitInlineLink(t *testing.T) {
	opened := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			marketReadme(c, Model{OnOpenURL: func(url string) { opened = url }}, readmeFixture,
				c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
		})
	}, 640, 1000)
	if opened != "" {
		t.Fatal("document rendering activated an external URL")
	}
	for _, text := range []string{"Native project tools", "structured documentation", "project.json", "Inspect", "Read only", "Relative illustration", "Remote illustration", "<b>", "</b>"} {
		if !tt.HasText(text) {
			t.Errorf("native document lost %q", text)
		}
	}
	if tt.HasText("# Native project tools") || tt.HasText("**structured documentation**") || tt.HasText("![Relative illustration]") {
		t.Fatal("README still rendered as raw Markdown text")
	}
	if _, ok := tt.Find("Copy code"); !ok {
		t.Fatal("README did not use native fenced-code controls")
	}
	if err := tt.Click("Project guide"); err != nil {
		t.Fatal(err)
	}
	if opened != "https://example.invalid/guide" {
		t.Fatalf("inline documentation link opened %q", opened)
	}
	if err := tt.Click("Copy code"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(tt.Clipboard(), "{\"workspace\":\"fixture\"") {
		t.Fatalf("native code copy=%q", tt.Clipboard())
	}
}

func TestMarketReadmeEmptyDoesNotMountSection(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		marketReadme(c, Model{}, "  \n", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
	}, 400, 400)
	if _, ok := tt.Find("Repository README"); ok {
		t.Fatal("empty details mounted an empty README section")
	}
}

func TestMarketReadmeSourceHeadingParagraphAndQuoteGeometry(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	for _, size := range []float32{14, 18} {
		prefs := previous
		prefs.Chat.Size = size
		theme.SetPreferences(prefs)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				marketReadme(c, Model{}, "# Source heading\n\nFirst paragraph\n\nSecond paragraph\n\n> Quoted paragraph", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
			})
		}, 640, 720)
		find := func(want string) ui.Rect {
			t.Helper()
			for _, text := range tt.Texts() {
				if strings.TrimSpace(text) == want {
					if r, ok := tt.Find(text); ok {
						return r
					}
				}
			}
			t.Fatalf("missing native text %q", want)
			return ui.Rect{}
		}
		heading := find("Source heading")
		first, second, quote := find("First paragraph"), find("Second paragraph"), find("Quoted paragraph")
		delta := size - 14
		if heading.H != 30+delta || first.H != 24+delta || second.H != 24+delta || quote.H != 24+delta ||
			first.Y-heading.Y-heading.H != 16 || second.Y-first.Y-first.H != 16 || quote.Y-second.Y-second.H != 16 || quote.X-first.X != 16 {
			t.Fatalf("source typography size%v heading=%+v first=%+v second=%+v quote=%+v", size, heading, first, second, quote)
		}
	}
}

func TestMarketReadmeSourceListAndRuleGeometry(t *testing.T) {
	previous := theme.CurrentPreferences()
	prefs := previous
	prefs.Chat.Size = 14
	theme.SetPreferences(prefs)
	t.Cleanup(func() { theme.SetPreferences(previous) })
	for _, loose := range []bool{false, true} {
		source := "Before list\n\n- First item\n  - Nested first\n  - Nested second\n- Second item\n\nAfter list\n\n---\n\nAfter rule"
		if loose {
			source = "- First paragraph\n\n  Second paragraph\n\n- Next item"
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				marketReadme(c, Model{}, source, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
			})
		}, 640, 1000)
		find := func(want string) ui.Rect {
			t.Helper()
			for _, text := range tt.Texts() {
				if strings.TrimSpace(text) == want {
					if r, ok := tt.Find(text); ok {
						return r
					}
				}
			}
			t.Fatalf("missing native text %q", want)
			return ui.Rect{}
		}
		if loose {
			first, second, next := find("First paragraph"), find("Second paragraph"), find("Next item")
			if second.Y-first.Y-first.H != 8 || next.Y-second.Y-second.H != 6 || first.X != 18 {
				t.Fatalf("loose-list paragraph/item margins first=%+v second=%+v next=%+v", first, second, next)
			}
			continue
		}
		before, first := find("Before list"), find("First item")
		nestedFirst, nestedSecond := find("Nested first"), find("Nested second")
		second, after := find("Second item"), find("After list")
		rule, ok := tt.Find("Markdown thematic break")
		afterRule := find("After rule")
		if first.X-before.X != 18 || nestedFirst.X-first.X != 18 || first.Y-before.Y-before.H != 16 ||
			nestedFirst.Y-first.Y-first.H != 4 || nestedSecond.Y-nestedFirst.Y-nestedFirst.H != 6 ||
			second.Y-nestedSecond.Y-nestedSecond.H != 6 || after.Y-second.Y-second.H != 16 ||
			!ok || rule.H != .5 || rule.Y-after.Y-after.H != 32 || afterRule.Y-rule.Y-rule.H != 32 {
			t.Fatalf("source list/rule geometry: before=%+v first=%+v nested1=%+v nested2=%+v second=%+v after=%+v rule=%+v last=%+v", before, first, nestedFirst, nestedSecond, second, after, rule, afterRule)
		}
	}
}

func TestMarketReadmeH3CascadeKeepsAbsoluteMetricsAndSharedMargins(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	for _, size := range []float32{14, 18} {
		prefs := previous
		prefs.Chat.Size = size
		theme.SetPreferences(prefs)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				marketReadme(c, Model{}, "# Level one\n\n## Level two\n\n### Level three\n\nBody after heading\n\n#### Level four\n\n> ### Nested three\n>\n> Quoted body", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
			})
		}, 360, 900)
		find := func(want string) ui.Rect {
			t.Helper()
			for _, text := range tt.Texts() {
				if strings.TrimSpace(text) == want {
					if r, ok := tt.Find(text); ok {
						return r
					}
				}
			}
			t.Fatalf("missing native text %q", want)
			return ui.Rect{}
		}
		one, two, three := find("Level one"), find("Level two"), find("Level three")
		body, four := find("Body after heading"), find("Level four")
		nested, quoted := find("Nested three"), find("Quoted body")
		delta := size - 14
		if one.H != 30+delta || two.H != 28+delta || three.H != 22 || nested.H != 22 || four.H != 24+delta || body.H != 24+delta ||
			three.Y-two.Y-two.H != 32 || body.Y-three.Y-three.H != 16 || quoted.Y-nested.Y-nested.H != 16 || nested.X-body.X != 16 {
			t.Fatalf("Market h3 cascade at content size%v: h1=%+v h2=%+v h3=%+v body=%+v h4=%+v nested=%+v quoted=%+v", size, one, two, three, body, four, nested, quoted)
		}
	}
}

func TestMarketReadmeH3CascadeWrapsAtFourteenPixels(t *testing.T) {
	previous := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previous) })
	const heading = "Review repository documentation before changing project configuration"
	for _, size := range []float32{14, 18} {
		prefs := previous
		prefs.Chat.Size = size
		theme.SetPreferences(prefs)
		for _, width := range []int{150, 360} {
			reference := ui.NewTester(func(c *ui.Context) {
				ui.RichText(c, ui.Span{Text: heading}).Font(prefs.Chat.Family).FontSize(14).FontWeight(700).FixedLineHeight(22)
			}, width, 500)
			want, ok := reference.Find(heading)
			if !ok {
				t.Fatal("missing reference heading")
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					marketReadme(c, Model{}, "### "+heading, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
				})
			}, width, 500)
			got, ok := tt.Find(heading)
			if !ok || got.W != want.W || got.H != want.H || (width == 150 && got.H <= 22) {
				t.Fatalf("Market h3 wrap at size%v width%d: got=%+v want=%+v", size, width, got, want)
			}
		}
	}
}

func TestMarketRichReadmeDetailsWideAndNarrow(t *testing.T) {
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	item := desktopmarket.CatalogItem{ID: "fixture/native-tools", Repo: "Native tools", Owner: "fixture", Description: "Offline native README fixture", Homepage: "https://example.invalid/fixture/native-tools", InstallSpec: "fixture-native-tools"}
	d := marketstore.Discovery{Selected: &item, Details: desktopmarket.Details{Readme: readmeFixture, Version: "1.2.3"}}
	for _, size := range [][2]int{{1000, 1000}, {420, 720}} {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Background(c.Theme().Background)
			catalogDetailsDialog(c, Model{Store: store}, d, marketstore.ViewState{}, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Border, c.Theme().Accent)
		}, size[0], size[1])
		panel, ok := tt.Find("Native tools")
		section, mounted := tt.Find("Repository README")
		close, closeOK := tt.Find(marketText(i18n.T(i18n.KeyW4MarketClose), "Close"))
		if !ok || !mounted || !closeOK || panel.W > 680 || panel.W > float32(size[0])-32 || section.W > panel.W-32 || close.X < 0 || close.Y < 0 || close.X+close.W > float32(size[0]) || close.Y+close.H > float32(size[1]) {
			t.Fatalf("size%v escaped native details: panel=%+v section=%+v close=%+v", size, panel, section, close)
		}
		name := "market-rich-readme-wide.png"
		if size[0] == 420 {
			name = "market-rich-readme-narrow.png"
		}
		saveMarketSnapshot(t, tt, name)
	}
}
