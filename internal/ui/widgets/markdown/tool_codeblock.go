package markdown

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// ToolCodeBlock draws the shared CodeBlock used by run_code. Its owner supplies
// the scroll viewport and margin; the banner sticks to that viewport's top.
// Copy follows the displayed preformatted text, trimming one final newline.
func ToolCodeBlock(c *ui.Context, language, code string, st Style, scrollY float32) ui.Element {
	code = strings.TrimSuffix(code, "\n")
	font := theme.RenderPreferences().Code.Family
	languageInk := st.ToolCodeLanguage
	if languageInk.A == 0 {
		languageInk = st.Muted
	}
	root := ui.Column(c.Key("code-card")).MinWidth(0).Shrink(0).Background(st.CodeBG).Radius(16)
	wrap := ui.Local(root, "wrap-code", func() bool { return true })
	copiedUntil := ui.Local(root, "copied-until", func() time.Time { return time.Time{} })
	copied := c.Now().Before(*copiedUntil)
	if copied {
		c.After(copiedUntil.Sub(c.Now()))
	}
	// A flow spacer reserves42px while the overlaid banner follows the owner's
	// scroll offset, keeping the source in its original document position.
	root.Children(func() {
		ui.Box(c).Height(42).Shrink(0)
		spans := toolCodeSpans(language, code, st)
		paint := func() {
			text := ui.RichText(c, spans...).Font(font).FontSize(11).FixedLineHeight(19).TextColor(st.FG).
				MinWidth(0).Shrink(0).Selectable().Padding(6, 22, 20, 22)
			if !*wrap {
				text.NoWrap()
			}
		}
		if *wrap {
			paint()
		} else {
			ui.ScrollHorizontal(c.Key("columns")).MinWidth(0).Children(paint)
		}
		top := max(float32(0), scrollY-4)
		// A sticky banner cannot leave the bottom of its code-card ancestor.
		bounds := ui.Local(root, "bounds", func() ui.Rect { return ui.Rect{} })
		root.ObserveLayout(func(r ui.Rect) { *bounds = r })
		if bounds.H > 0 {
			top = min(top, max(float32(0), bounds.H-42))
		}
		ui.Row(c.Key("banner")).Absolute().Top(top).Left(0).Right(0).Height(42).Padding(10, 18, 8, 22).
			Gap(12).AlignItems(ui.Center).Background(st.CodeBG).Radius(16, 16, 0, 0).Label(codeLabel(st)).Children(func() {
			ui.Text(c, language).Font(font).FontSize(11).FixedLineHeight(18).TextColor(languageInk).NoWrap().Grow(1).MinWidth(0)
			ui.Row(c).Gap(4).Shrink(0).Children(func() {
				wrapLabel, unwrapLabel := codeWrapLabels(st)
				icon, tooltip := codeIconWrap, wrapLabel
				if *wrap {
					icon, tooltip = codeIconNoWrap, unwrapLabel
				}
				button := toolCodeButton(c.Key("wrap"), icon, wrapLabel, tooltip, st).Role(ui.RoleToggleButton).Checked(*wrap)
				if button.Clicked() {
					*wrap = !*wrap
					c.Invalidate()
				}
				icon, label := codeIconCopy, codeCopyLabel(st)
				if copied {
					icon, label = codeIconCheck, codeCopiedLabel(st)
				}
				if toolCodeButton(c.Key("copy"), icon, label, label, st).Clicked() && !copied {
					c.WriteClipboard(code)
					*copiedUntil = c.Now().Add(time.Second)
					c.Invalidate()
				}
			})
		})
	})
	return root
}

func toolCodeButton(c *ui.Context, icon *ui.SVG, label, tooltip string, st Style) ui.Element {
	button := ui.ButtonBase(c).Size(24, 24).Padding(0).Radius(8).Label(label).Tooltip(tooltip).
		Background(ui.Transparent).TextColor(st.Muted)
	if button.Hovered() {
		hover := st.ToolCodeHover
		if hover.A == 0 {
			hover = st.Muted.Alpha(.12)
		}
		button.Background(hover)
	}
	button.Children(func() { ui.Icon(c, icon).Size(14, 14).PassThrough() })
	return button
}
