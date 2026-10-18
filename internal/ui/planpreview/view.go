// Package planpreview renders read-only plan Markdown without opening a webview
// or executing anything embedded in the reviewed document.
package planpreview

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// Style uses the current conversation colors.
type Style struct{ FG, Muted, Accent, Field, Border ui.Color }

// View presents the complete document in an independently scrollable reader.
// Only disclosure state is UI-local; document contents always come from the
// current session snapshot or exact pending request, never a global cache.
func View(c *ui.Context, key, title, source string, initiallyOpen bool, style Style) {
	box := ui.Column(c.Key(key)).FillWidth().Gap(6).Padding(10).Background(style.Field).Border(1, style.Border).Radius(8)
	open := ui.Local(box, "plan-document-open", func() bool { return initiallyOpen })
	box.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(style.FG).Grow(1)
			label := Text("阅读计划", "Read plan")
			if *open {
				label = Text("收起文档", "Collapse document")
			}
			if ui.Button(c, label).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
				*open = !*open
			}
			if ui.Button(c, Text("复制文档", "Copy document")).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
				mygo.Clipboard.WriteText(source)
			}
		})
		ui.Text(c, Text("只读计划文档", "Read-only plan document")).FontSize(fonttheme.InterfaceSize(11)).TextColor(style.Muted)
		if *open {
			ui.Scroll(c.Key(key + "-reader")).FillWidth().Height(280).Children(func() {
				ui.Column(c).FillWidth().Padding(8).Children(func() {
					markdown.View(c, source, markdown.Style{FontSize: 14, FG: style.FG, Muted: style.Muted, Accent: style.Accent, CodeBG: style.Field})
				})
			})
		}
	})
}

// Text selects the active UI locale for the small standalone plan surface.
func Text(zh, en string) string {
	if i18n.Locale() == i18n.LocaleZH {
		return zh
	}
	return en
}
