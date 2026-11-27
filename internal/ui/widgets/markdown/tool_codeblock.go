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
	root := ui.Column(c.Key("code-card")).MinWidth(0).Shrink(0).Radius(16).Clip()
	if st.CodeSurface != nil {
		root = st.CodeSurface(root, 16)
	} else {
		root = root.Background(st.CodeBG)
	}
	wrap := ui.Local(root, "wrap-code", func() bool { return true })
	copiedUntil := ui.Local(root, "copied-until", func() time.Time { return time.Time{} })
	copied := c.Now().Before(*copiedUntil)
	if copied {
		c.After(copiedUntil.Sub(c.Now()))
	}
	rawTop := max(float32(0), scrollY-4)
	clampedTop := func(h float32) float32 {
		if h > 0 {
			return min(rawTop, max(float32(0), h-42))
		}
		return rawTop
	}
	// A sticky banner cannot leave the bottom of its code-card ancestor.
	bounds := ui.Local(root, "bounds", func() ui.Rect { return ui.Rect{} })
	inv := c.Services().Invalidate
	root.ObserveLayout(func(r ui.Rect) {
		prevH := bounds.H
		*bounds = r
		if r.H != prevH && rawTop > max(float32(0), r.H-42) && inv != nil {
			inv()
		}
	})
	// A flow spacer reserves 42px while the overlaid banner follows the owner's
	// scroll offset, keeping the source in its original document position.
	// When scrolled (top > 0), shifting codeClip down by +top and codeContent
	// back up by -top clips code lines cleanly below the sticky banner without
	// changing card height or painting an opaque patch over translucent glass.
	// Applying the clamped displacement in OffsetAfterLayout (after ObserveLayout
	// writes bounds.H on the same frame) prevents a blank first frame when a
	// short block mounts with a large scrollY.
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
		codeClip := ui.Box(c.Key("code-clip")).FillWidth().MinWidth(0).Shrink(0).Clip()
		if rawTop > 0 {
			codeClip.OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{Y: clampedTop(bounds.H)} })
		}
		codeClip.Children(func() {
			codeContent := ui.Box(c.Key("code-content")).FillWidth().MinWidth(0).Shrink(0)
			if rawTop > 0 {
				codeContent.OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{Y: -clampedTop(bounds.H)} })
			}
			codeContent.Children(func() {
				if *wrap {
					paint()
				} else {
					ui.ScrollHorizontal(c.Key("columns")).MinWidth(0).Children(paint)
				}
			})
		})
		banner := ui.Row(c.Key("banner")).Absolute().Top(0).Left(0).Right(0).Height(42).Padding(10, 18, 8, 22).
			Gap(12).AlignItems(ui.Center).Radius(16, 16, 0, 0).Label(codeLabel(st))
		if rawTop > 0 {
			banner.OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{Y: clampedTop(bounds.H)} })
		}
		// Keep banner transparent in both opaque and glass modes: root paints
		// the card surface while codeClip prevents scrolled lines from bleeding
		// behind the sticky controls. Draw a 1px hairline when stuck.
		if rawTop > 0 {
			divider := st.Muted.Alpha(0.16)
			banner.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if clampedTop(bounds.H) <= 0 || r.W <= 0 || r.H <= 0 || divider.A == 0 {
					return
				}
				px := float32(1)
				if scale := p.Scale(); scale > 0 {
					px = 1 / scale
				}
				p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - px, W: r.W, H: px}, divider, 0)
			})
		}
		banner.Children(func() {
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
