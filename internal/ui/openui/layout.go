package openui

import (
	"math"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

func init() {
	Register("Col", colView)
	Register("Row", rowView)
	Register("Box", boxView)
	Register("Grid", gridView)
	Register("Card", cardView)
	Register("Carousel", carouselView)
	Register("Divider", func(r *Render, el *lang.Element) ui.Element { return coss.Separator(r.C, coss.Horizontal) })
	Register("Spacer", func(r *Render, el *lang.Element) ui.Element { return ui.Box(r.C).Grow(1) })
	Register("Details", detailsView)
	Register("Popover", popoverView)
	Register("Form", func(r *Render, el *lang.Element) ui.Element {
		return ui.Column(r.C).FillWidth().MinWidth(0).Gap(design.Sp(3)).Role(ui.RoleGroup).Children(func() { r.Children(el, "children") })
	})
}

func colView(r *Render, el *lang.Element) ui.Element {
	return ui.Column(r.C).FillWidth().MinWidth(0).Gap(spacing(numOr(el, "gap", 3))).
		AlignItems(align(str(el, "align"), ui.Stretch)).
		Children(func() { r.Children(el, "children") })
}

func rowView(r *Render, el *lang.Element) ui.Element {
	row := ui.Row(r.C).FillWidth().MinWidth(0).Gap(spacing(numOr(el, "gap", 2))).
		AlignItems(align(str(el, "align"), ui.Center)).Justify(align(str(el, "justify"), ui.Start))
	if boolean(el, "wrap") {
		row.Wrap()
	}
	return row.Children(func() { r.Within(true, func() { r.Content(el.Prop("children")) }) })
}

func boxView(r *Render, el *lang.Element) ui.Element {
	var b ui.Element
	if str(el, "direction") == "row" {
		b = ui.Row(r.C).AlignItems(align(str(el, "align"), ui.Center))
	} else {
		b = ui.Column(r.C).AlignItems(align(str(el, "align"), ui.Stretch))
	}
	b.MinWidth(0).Gap(spacing(numOr(el, "gap", 0))).Justify(align(str(el, "justify"), ui.Start))
	if p, ok := num(el, "padding"); ok {
		b.Padding(spacing(p))
	}
	if bg, ok := r.softColor(str(el, "background")); ok {
		b.Background(bg)
	}
	if el.Prop("radius") != nil || el.Prop("background") != nil || boolean(el, "border") {
		b.Radius(radius(el.Prop("radius"), design.RadiusLG))
	}
	if boolean(el, "border") {
		b.Border(design.BorderWidth, r.T.Border)
	}
	if w, ok := pixels(el.Prop("width")); ok {
		b.Width(w).Shrink(0)
	} else if s, _ := el.Prop("width").(string); s == "100%" {
		b.FillWidth()
	}
	if h, ok := pixels(el.Prop("height")); ok {
		b.Height(h)
	}
	if f, ok := num(el, "flex"); ok && f > 0 {
		b.Grow(float32(min(f, 12)))
	}
	if boolean(el, "wrap") {
		b.Wrap()
	}
	isRow := str(el, "direction") == "row"
	return b.Children(func() { r.Within(isRow, func() { r.Content(el.Prop("children")) }) })
}

type gridMemo struct{ width float32 }

func gridView(r *Render, el *lang.Element) ui.Element {
	g := ui.Grid(r.C).FillWidth().MinWidth(0)
	gap := spacing(numOr(el, "gap", 3))
	g.Gap(gap)
	memo := ui.Local(g, "openui.grid", func() gridMemo { return gridMemo{} })
	kids := el.Children()
	cols := int(numOr(el, "columns", 0))
	if minW, ok := num(el, "minChildWidth"); ok && minW > 0 {
		width := memo.width
		if width <= 0 {
			width = r.opts.Width
		}
		fit := 1
		if width > 0 {
			fit = max(1, int(math.Floor(float64(width+gap)/(minW+float64(gap)))))
		}
		if cols <= 0 || fit < cols {
			cols = fit
		}
		// Balanced rows: 4 items in 3 columns become 2×2.
		if n := len(kids); n > cols && cols > 1 {
			rows := (n + cols - 1) / cols
			cols = (n + rows - 1) / rows
		}
	}
	if cols <= 0 {
		cols = min(max(1, len(kids)), 2)
	}
	cols = min(cols, 12)
	g.Columns(cols)
	g.ObserveLayout(func(b ui.Rect) { memo.width = b.W })
	return g.Children(func() {
		for i, k := range kids {
			// A cell column gives FillWidth children their cell as the parent.
			r.C.Key(i)
			ui.Column(r.C).MinWidth(0).Children(func() { r.Within(false, func() { r.Node(k) }) })
		}
	})
}

func cardView(r *Render, el *lang.Element) ui.Element {
	t := r.T
	card := ui.Column(r.C).FillWidth().MinWidth(0).Radius(design.RadiusXL).Gap(spacing(numOr(el, "gap", 3))).
		Padding(spacing(numOr(el, "padding", 5)))
	if str(el, "variant") == "soft" {
		card.Background(t.Muted)
	} else {
		card.Background(t.Card).Border(design.BorderWidth, t.Border)
		s := design.ShadowXs[0]
		card.Shadow(s.X, s.Y, s.Blur, s.Spread, s.Color)
	}
	return card.Children(func() { r.Children(el, "children") })
}

func carouselView(r *Render, el *lang.Element) ui.Element {
	w := float32(numOr(el, "itemWidth", 260))
	w = max(120, min(w, 640))
	return coss.ScrollArea(r.C, coss.ScrollAreaOptions{Horizontal: true}, func() {
		ui.Row(r.C).Gap(spacing(numOr(el, "gap", 3))).AlignItems(ui.Stretch).PaddingY(design.Sp(1)).Children(func() {
			for i, k := range el.Children() {
				r.C.Key(i)
				ui.Column(r.C).Width(w).Shrink(0).Children(func() { r.Node(k) })
			}
		})
	})
}

func detailsView(r *Render, el *lang.Element) ui.Element {
	box := ui.Column(r.C).FillWidth().MinWidth(0)
	open := ui.Local(box, "openui.details", func() bool { return boolean(el, "defaultOpen") })
	t := r.T
	return box.Children(func() {
		coss.Collapsible(r.C, open, func(isOpen bool) {
			ui.Row(r.C).Gap(design.Sp(1.5)).AlignItems(ui.Center).PaddingY(design.Sp(1)).Children(func() {
				rot := float32(0)
				if isOpen {
					rot = 90
				}
				r.icon("chevron-right", design.IconSM, t.MutedForeground).Rotate(rot)
				r.text(str(el, "label"), design.TextSM, design.WeightMedium, t.Foreground)
			})
		}, func() {
			ui.Column(r.C).FillWidth().Gap(design.Sp(3)).Padding(design.Sp(2), 0, design.Sp(1), design.Sp(5.5)).Children(func() {
				r.Children(el, "children")
			})
		})
	})
}

func popoverView(r *Render, el *lang.Element) ui.Element {
	box := ui.Row(r.C)
	open := ui.Local(box, "openui.popover", func() bool { return false })
	r.Inline(box)
	return box.Children(func() {
		var anchor ui.Element
		if s, ok := el.Prop("trigger").(string); ok {
			anchor = coss.Button(r.C, s, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM})
		} else {
			anchor = ui.ButtonBase(r.C).Cursor(ui.CursorPointer).Children(func() { r.Content(el.Prop("trigger")) })
		}
		if anchor.Clicked() {
			*open = !*open
		}
		coss.Popover(r.C, anchor, open, coss.PopoverOptions{Width: 280}, func() {
			ui.Column(r.C).FillWidth().Gap(design.Sp(2)).Children(func() { r.Content(el.Prop("content")) })
		})
	})
}
