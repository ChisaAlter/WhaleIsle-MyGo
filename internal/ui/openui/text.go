package openui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

func init() {
	Register("Title", titleView)
	Register("Text", textView)
	Register("Caption", func(r *Render, el *lang.Element) ui.Element {
		col := r.T.MutedForeground
		if c, ok := r.color(str(el, "color")); ok {
			col = c
		}
		return r.paragraph(str(el, "text"), design.TextXS, design.WeightNormal, col)
	})
	Register("Label", func(r *Render, el *lang.Element) ui.Element {
		return r.paragraph(str(el, "text"), design.TextSM, design.WeightMedium, r.T.Foreground)
	})
	Register("Badge", badgeView)
	Register("Link", linkView)
	Register("Markdown", func(r *Render, el *lang.Element) ui.Element {
		return r.markdown(str(el, "text"))
	})
	Register("Quote", quoteView)
	Register("Code", codeView)
	Register("List", listView)
	Register("ListItem", func(r *Render, el *lang.Element) ui.Element { return listItem(r, el, "", false, false) })
}

// markdownStyle completes the transcript style with the design tokens.
func (r *Render) markdownStyle() markdown.Style {
	st := r.opts.Markdown
	t := r.T
	if st.FG.A == 0 {
		st.FG = t.Foreground
	}
	if st.Muted.A == 0 {
		st.Muted = t.MutedForeground
	}
	if st.Accent.A == 0 {
		st.Accent = t.Primary
	}
	if st.CodeBG.A == 0 {
		st.CodeBG = t.Muted
	}
	if st.FontSize <= 0 {
		st.FontSize = design.TextSM.Size
	}
	if !st.Dark {
		st.Dark = t.IsDark()
	}
	st.RenderFence = nil
	st.Streaming = r.Frame.Streaming
	st.OnOpenURL = r.opts.OpenURL
	st.CopyLabel = i18n.Pick("复制", "Copy")
	st.CopiedLabel = i18n.Pick("已复制", "Copied")
	return st
}

// markdown renders block Markdown (headings, lists, quotes, fenced code
// with the app's highlighter, tables) in the card's prose style.
func (r *Render) markdown(src string) ui.Element {
	box := ui.Column(r.C).FillWidth().MinWidth(0)
	box.Children(func() { markdown.View(r.C, src, r.md) })
	return box
}

func titleView(r *Render, el *lang.Element) ui.Element {
	ts := textStyle(str(el, "size"), design.TextBase)
	if str(el, "size") == "md" || str(el, "size") == "" {
		ts = design.TextLG
	}
	col := r.T.Foreground
	if c, ok := r.color(str(el, "color")); ok {
		col = c
	}
	w := design.WeightSemibold
	if ts.Size >= design.TextXL2.Size {
		w = design.WeightBold
	}
	return r.paragraph(str(el, "text"), ts, w, col).Role(ui.RoleHeading)
}

func textView(r *Render, el *lang.Element) ui.Element {
	ts := textStyle(str(el, "size"), design.TextSM)
	col := r.T.Foreground
	if c, ok := r.color(str(el, "color")); ok {
		col = c
	}
	w := weight(str(el, "weight"), design.WeightNormal)
	if arr, ok := el.Prop("text").([]lang.Value); ok {
		// Mixed arrays may contain components: lay them out inline.
		return ui.Row(r.C).Wrap().GapX(design.Sp(1)).AlignItems(ui.Center).MinWidth(0).Children(func() {
			for i, it := range arr {
				r.C.Key(i)
				if s, ok := it.(string); ok {
					r.inlineText(s, ts, w, col, el)
				} else {
					r.Content(it)
				}
			}
		})
	}
	return r.inlineText(str(el, "text"), ts, w, col, el)
}

// inlineText renders a paragraph with inline bold, italic, code and links.
func (r *Render) inlineText(s string, ts design.TextStyle, w int, col ui.Color, el *lang.Element) ui.Element {
	st := r.md
	st.FontSize, st.FG = ts.Size, col
	var e ui.Element
	if strings.ContainsAny(s, "*_`[~") {
		e = ui.RichText(r.C, markdown.Spans(s, st)...)
	} else {
		e = ui.Text(r.C, s)
	}
	e.FontSize(ts.Size).FixedLineHeight(ts.LineHeight).FontWeight(w).TextColor(col).Selectable().MinWidth(0)
	if el != nil {
		if boolean(el, "italic") {
			e.Italic()
		}
		if boolean(el, "lineThrough") {
			e.Strikethrough()
		}
		if boolean(el, "tabularNums") {
			e.FontFeatures("tnum")
		}
		if n, ok := num(el, "maxLines"); ok && n >= 1 {
			e.MaxLines(int(min(n, 100)))
		}
	}
	return e
}

func badgeView(r *Render, el *lang.Element) ui.Element {
	variant := coss.BadgeSecondary
	switch str(el, "color") {
	case "info":
		variant = coss.BadgeInfo
	case "success":
		variant = coss.BadgeSuccess
	case "warning":
		variant = coss.BadgeWarning
	case "danger":
		variant = coss.BadgeError
	case "purple", "pink":
		variant = coss.BadgeOutline
	}
	label := str(el, "text")
	if name := str(el, "icon"); name != "" {
		return r.Inline(ui.Row(r.C)).Gap(design.Sp(1)).AlignItems(ui.Center).Children(func() {
			r.icon(name, design.IconXS, r.T.MutedForeground)
			coss.Badge(r.C, label, coss.BadgeOptions{Variant: variant})
		})
	}
	return r.Inline(coss.Badge(r.C, label, coss.BadgeOptions{Variant: variant}))
}

func linkView(r *Render, el *lang.Element) ui.Element {
	label, href := str(el, "text"), str(el, "url")
	if label == "" {
		label = href
	}
	if r.opts.OpenURL == nil || lang.ValidateExternalURL(href) != nil {
		return r.paragraph(label, design.TextSM, design.WeightNormal, r.T.Foreground)
	}
	b := r.Inline(ui.ButtonBase(r.C)).Cursor(ui.CursorPointer).Role(ui.RoleLink).Tooltip(href).FocusRing(true)
	b.Children(func() {
		r.text(label, design.TextSM, design.WeightMedium, r.T.Primary).Underline()
	})
	if b.Clicked() {
		r.opts.OpenURL(href)
	}
	return b
}

func quoteView(r *Render, el *lang.Element) ui.Element {
	t := r.T
	return ui.Column(r.C).FillWidth().Gap(design.Sp(1)).PaddingX(design.Sp(3.5)).BorderWidth(0, 0, 0, 2).BorderColor(t.Border).Children(func() {
		r.inlineText(str(el, "text"), design.TextSM, design.WeightNormal, t.Foreground, nil).Italic()
		if cite := str(el, "cite"); cite != "" {
			r.paragraph("— "+cite, design.TextXS, design.WeightNormal, t.MutedForeground)
		}
	})
}

// fence wraps code in a Markdown fence longer than any backtick run in it.
func fence(language, code string) string {
	n, run := 3, 0
	for _, ch := range code {
		if ch == '`' {
			run++
			n = max(n, run+1)
		} else {
			run = 0
		}
	}
	f := strings.Repeat("`", n)
	return f + strings.Fields(language + " ")[0] + "\n" + strings.TrimRight(code, "\n") + "\n" + f
}

func codeView(r *Render, el *lang.Element) ui.Element {
	language := strings.TrimSpace(str(el, "language"))
	if language == "" {
		language = "text"
	}
	return r.markdown(fence(language, str(el, "code")))
}

func listView(r *Render, el *lang.Element) ui.Element {
	marker := str(el, "marker")
	timeline := boolean(el, "timeline")
	items := list(el, "items")
	return ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(timelineGap(timeline))).Role(ui.RoleList).Children(func() {
		for i, it := range items {
			r.C.Key(i)
			m := ""
			switch marker {
			case "number":
				m = lang.NumberToString(float64(i+1)) + "."
			case "none":
			default:
				m = "•"
			}
			last := i == len(items)-1
			switch t := it.(type) {
			case *lang.Element:
				if t.Type == "ListItem" {
					r.depth++
					listItem(r, t, m, timeline, last)
					r.depth--
				} else {
					r.Node(t)
				}
			default:
				listItem(r, &lang.Element{Type: "ListItem", Props: lang.ObjectFrom("title", lang.ToString(t))}, m, timeline, last)
			}
		}
	})
}

func timelineGap(timeline bool) float32 {
	if timeline {
		return 0
	}
	return 1.5
}

// listItem draws one item: marker (or the timeline dot and connector),
// title, description and children.
func listItem(r *Render, el *lang.Element, marker string, timeline, last bool) ui.Element {
	t := r.T
	if m := str(el, "marker"); m != "" {
		marker = m
	}
	return ui.Row(r.C).FillWidth().MinWidth(0).Gap(design.Sp(2.5)).AlignItems(ui.Stretch).Role(ui.RoleListItem).Children(func() {
		if timeline {
			ui.Column(r.C).Width(design.Sp(4)).Shrink(0).AlignItems(ui.Center).Children(func() {
				dot := ui.Box(r.C).Size(design.Sp(2.5), design.Sp(2.5)).Radius(design.RadiusFull).Background(t.Primary).Margin(design.Sp(1.5), 0, design.Sp(1), 0)
				_ = dot
				if !last {
					ui.Box(r.C).Width(design.BorderWidth * 2).Grow(1).MinHeight(design.Sp(3)).Background(t.Border)
				}
			})
		} else if marker != "" {
			ui.Box(r.C).MinWidth(design.Sp(4)).Shrink(0).Children(func() {
				if _, ok := lucideIcon(marker); ok && strings.Contains(marker, "-") {
					r.icon(marker, design.IconSM, t.MutedForeground).Margin(design.Sp(0.5), 0, 0, 0)
				} else {
					r.text(marker, design.TextSM, design.WeightNormal, t.MutedForeground)
				}
			})
		}
		col := ui.Column(r.C).Grow(1).MinWidth(0).Gap(design.Sp(0.5))
		if timeline && !last {
			col.Padding(0, 0, design.Sp(4), 0)
		}
		col.Children(func() {
			title := str(el, "title")
			if title != "" {
				w := design.WeightNormal
				if str(el, "description") != "" || timeline {
					w = design.WeightMedium
				}
				r.inlineText(title, design.TextSM, w, t.Foreground, nil)
			}
			if d := str(el, "description"); d != "" {
				r.inlineText(d, design.TextSM, design.WeightNormal, t.MutedForeground, nil)
			}
			r.Children(el, "children")
		})
	})
}
