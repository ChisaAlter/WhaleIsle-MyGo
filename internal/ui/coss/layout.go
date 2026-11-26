package coss

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// TabsVariant is coss's tabs list variant.
type TabsVariant uint8

const (
	// TabsDefault is the segmented look: a muted strip with a raised pill.
	TabsDefault TabsVariant = iota
	// TabsUnderline marks the chosen tab with a 2 DIP primary line.
	TabsUnderline
)

// TabsOptions configure Tabs.
type TabsOptions struct {
	Variant TabsVariant
	Size    SegmentedSize
	Icons   []Icon
	Label   string
	Key     any
}

// Tabs is coss's tab list choosing *selected; build the chosen panel below
// it. Arrows, Home and End move among tabs (TabsBase).
func Tabs(c *ui.Context, selected *int, labels []string, o TabsOptions) ui.Element {
	t := tok(c)
	withKey(c, o.Key)
	parts := ui.TabsBase(c, selected, len(labels))
	list := parts.List.Gap(design.Sp(0.5)).AlignItems(ui.Center)
	if o.Label != "" {
		list.Label(o.Label)
	}
	h, padX := float32(30), design.Sp(2.5)-1
	switch o.Size {
	case SegmentedSM:
		h, padX = 26, design.Sp(2)-1
	case SegmentedLG:
		h, padX = 34, design.Sp(3)-1
	}
	if o.Variant == TabsDefault {
		list.Radius(design.RadiusLG).Background(t.Muted).Padding(design.Sp(0.5))
	} else {
		list.PaddingY(design.Sp(1))
	}
	pill := t.Background
	if t.IsDark() {
		pill = t.Input
	}
	list.Children(func() {
		for i, l := range labels {
			tab := parts.Tab(i).Height(h).PaddingX(padX).Gap(design.Sp(1.5)).Radius(design.RadiusMD).Grow(1).
				Border(design.BorderWidth, ui.Transparent).FocusRing(false).Cursor(ui.CursorPointer).Justify(ui.Center)
			s := interact(tab, false)
			if i == 1 {
				s |= segForce
			}
			on := i == *selected
			fg := t.MutedForeground
			if o.Variant == TabsDefault {
				fg = t.MutedForeground.Alpha(design.OpacityPlaceholder)
			}
			if s.Has(StateHover) {
				fg = t.MutedForeground
				if o.Variant == TabsUnderline {
					tab.Background(t.Accent)
				}
			}
			var sh shadowList
			if on {
				fg = t.Foreground
				if o.Variant == TabsDefault {
					tab.Background(pill)
					sh.scale(design.ShadowSm, black, 0.05)
				}
			}
			if s.Has(StateFocusVisible) {
				sh.add(0, 0, 0, 2, t.Ring)
			}
			finish(tab, &sh, nil)
			tab.Children(func() {
				if i < len(o.Icons) && o.Icons[i] != IconNone {
					icon(c, o.Icons[i], design.IconSM, fg, false).MarginX(-2)
				}
				text(c, l, design.TextSM, design.WeightMedium, fg)
				if on && o.Variant == TabsUnderline {
					ui.Box(c).Absolute().Left(-1).Right(-1).Bottom(-design.Sp(1) - 2).Height(2).Background(t.Primary)
				}
			})
		}
	})
	return list
}

// CardOptions configure a Card.
type CardOptions struct {
	Title, Description string
	// Action goes at the header's top right.
	Action func()
	// Footer builds the card's footer row.
	Footer func()
	Key    any
}

// Card is coss's Card: a 16 DIP-radius bordered card surface with a
// header (18/18 semibold title, muted description), panel and footer, 24
// DIPs of padding.
func Card(c *ui.Context, o CardOptions, panel func()) ui.Element {
	t := tok(c)
	withKey(c, o.Key)
	bw := glassBorderWidth(t)
	card := ui.Column(c).Radius(design.RadiusXL2).Background(t.Card).Border(bw, t.Border).TextColor(t.CardForeground).
		Transition(ui.ElementTransition{Duration: design.MotionPopover, Ease: design.EaseInOut, Colors: true})
	var sh shadowList
	sh.scale(design.ShadowXs, black, 0.05)
	d := deco{border: bw, radius: design.RadiusXL2}
	d.highlight(t, 0.04, 0.06)
	applyGlassElevation(t, &sh, &d, glassTierCard)
	finish(card, &sh, &d)
	card.Children(func() {
		hasHeader := o.Title != "" || o.Description != "" || o.Action != nil
		if hasHeader {
			bottom := design.Sp(6)
			if panel != nil {
				bottom = design.Sp(4)
			}
			ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(design.Sp(4)).Padding(design.Sp(6), design.Sp(6), bottom, design.Sp(6)).Children(func() {
				ui.Column(c).Grow(1).Gap(design.Sp(1.5)).Children(func() {
					if o.Title != "" {
						para(c, o.Title, design.TextStyle{Size: design.TextLG.Size, LineHeight: design.TextLG.Size}, design.WeightSemibold, t.CardForeground).Role(ui.RoleHeading)
					}
					if o.Description != "" {
						para(c, o.Description, design.TextSM, design.WeightNormal, t.MutedForeground)
					}
				})
				if o.Action != nil {
					ui.Row(c).Shrink(0).Children(o.Action)
				}
			})
		}
		if panel != nil {
			top, bottom := design.Sp(6), design.Sp(6)
			if hasHeader {
				top = 0
			}
			if o.Footer != nil {
				bottom = 0
			}
			ui.Column(c).FillWidth().Padding(top, design.Sp(6), bottom, design.Sp(6)).Children(panel)
		}
		if o.Footer != nil {
			top := design.Sp(6)
			if panel != nil {
				top = design.Sp(4)
			}
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(design.Sp(2)).Padding(top, design.Sp(6), design.Sp(6), design.Sp(6)).Children(o.Footer)
		}
	})
	return card
}

// Frame is coss's Frame: a muted/72 tray (radius 16, 4 DIPs of padding)
// holding FramePanels 4 DIPs apart.
func Frame(c *ui.Context, fn func()) ui.Element {
	t := tok(c)
	saved := framePrevPanel
	framePrevPanel = false
	f := ui.Column(c).Radius(design.RadiusXL2).Background(t.Muted.Alpha(0.72)).Padding(design.Sp(1)).Children(fn)
	framePrevPanel = saved
	return f
}

// framePrevPanel tells a FramePanel that a panel came just before it in the
// Frame being built (coss puts 4 DIPs between consecutive panels only).
var framePrevPanel bool

// FramePanel is a bordered background panel of a Frame (radius 14, 20 DIPs
// of padding, shadow-xs/5 and the highlight).
func FramePanel(c *ui.Context, fn func()) ui.Element {
	t := tok(c)
	p := ui.Column(c).FillWidth().Radius(design.RadiusXL).Background(t.Background).Border(design.BorderWidth, t.Border).Padding(design.Sp(5))
	if framePrevPanel {
		p.Margin(design.Sp(1), 0, 0, 0)
	}
	framePrevPanel = true
	var sh shadowList
	sh.scale(design.ShadowXs, black, 0.05)
	d := deco{border: design.BorderWidth, radius: design.RadiusXL}
	d.highlight(t, 0.04, 0.06)
	finish(p, &sh, &d)
	return p.Children(fn)
}

// FrameHeader titles a Frame section (14/20 semibold and muted, 16×20
// padding).
func FrameHeader(c *ui.Context, title, description string) ui.Element {
	t := tok(c)
	framePrevPanel = false
	return ui.Column(c).FillWidth().Padding(design.Sp(4), design.Sp(5)).Children(func() {
		if title != "" {
			para(c, title, design.TextSM, design.WeightSemibold, t.Foreground)
		}
		if description != "" {
			para(c, description, design.TextSM, design.WeightNormal, t.MutedForeground)
		}
	})
}

// AlertVariant is coss's alertVariants.variant.
type AlertVariant uint8

const (
	AlertDefault AlertVariant = iota
	AlertInfo
	AlertSuccess
	AlertWarning
	AlertError
)

// AlertOptions configure an Alert.
type AlertOptions struct {
	Variant AlertVariant
	// Icon defaults to the variant's (info, circle-check, triangle-alert,
	// circle-alert); IconNone with NoIcon set hides it.
	Icon   Icon
	NoIcon bool
	// Action builds buttons at the right, vertically centered.
	Action func()
	Key    any
}

// Alert is coss's inline Alert: a 14 DIP-radius bordered note with an
// icon, a medium title and a muted description. It is the app's feedback
// surface in place of toasts; description must be a sentence from
// internal/errcopy when it reports a failure.
func Alert(c *ui.Context, title, description string, o AlertOptions) ui.Element {
	t := tok(c)
	withKey(c, o.Key)
	bc, bg, ic := t.Border, ui.Transparent, t.MutedForeground
	glyph := o.Icon
	if t.IsDark() {
		bg = t.Input.Alpha(0.32)
	}
	tint := func(col ui.Color, def Icon) {
		bc, bg, ic = col.Alpha(0.32), col.Alpha(0.04), col
		if glyph == IconNone {
			glyph = def
		}
	}
	switch o.Variant {
	case AlertInfo:
		tint(t.Info, IconInfo)
	case AlertSuccess:
		tint(t.Success, IconCircleCheck)
	case AlertWarning:
		tint(t.Warning, IconTriangleAlert)
	case AlertError:
		tint(t.Destructive, IconCircleAlert)
	default:
		if glyph == IconNone {
			glyph = IconInfo
		}
	}
	if o.NoIcon {
		glyph = IconNone
	}
	a := ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(design.Sp(2)).Radius(design.RadiusXL).Border(design.BorderWidth, bc).Background(bg).
		Padding(design.Sp(3), design.Sp(3.5)).Role(ui.RoleStatus)
	a.Children(func() {
		if glyph != IconNone {
			ui.Box(c).Size(16, design.TextSM.LineHeight).Center().Shrink(0).Children(func() {
				icon(c, glyph, design.IconSM, ic, true)
			})
		}
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
			if title != "" {
				para(c, title, design.TextSM, design.WeightMedium, t.CardForeground)
			}
			if description != "" {
				para(c, description, design.TextSM, design.WeightNormal, t.MutedForeground)
			}
		})
		if o.Action != nil {
			ui.Row(c).Gap(design.Sp(1)).AlignSelf(ui.Center).Shrink(0).Children(o.Action)
		}
	})
	return a
}

// EmptyOptions configure an Empty.
type EmptyOptions struct {
	// Icon shows in coss's media tile (36 DIPs, bordered card).
	Icon        Icon
	Title       string
	Description string
	Key         any
}

// Empty is coss's Empty state: a centered column (24 DIPs apart, 24×80
// padding) with an icon tile, a 20/28 bold title, a muted description and
// actions in content.
func Empty(c *ui.Context, o EmptyOptions, content func()) ui.Element {
	t := tok(c)
	withKey(c, o.Key)
	return ui.Column(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Justify(ui.Center).Gap(design.Sp(6)).Padding(design.Sp(20), design.Sp(6)).Children(func() {
		ui.Column(c).MaxWidth(384).AlignItems(ui.Center).Children(func() {
			if o.Icon != IconNone {
				tile := ui.Box(c).Size(36, 36).Center().Radius(design.RadiusMD).Background(t.Card).Border(design.BorderWidth, t.Border).Margin(0, 0, design.Sp(6), 0)
				var sh shadowList
				sh.scale(design.ShadowSm, black, 0.05)
				d := deco{border: design.BorderWidth, radius: design.RadiusMD}
				d.highlight(t, 0.04, 0.06)
				finish(tile, &sh, &d)
				tile.Children(func() { icon(c, o.Icon, design.IconMD, t.Foreground, false) })
			}
			if o.Title != "" {
				para(c, o.Title, design.TextXL, design.WeightBold, t.Foreground).TextAlign(ui.Center).Role(ui.RoleHeading)
			}
			if o.Description != "" {
				d := para(c, o.Description, design.TextSM, design.WeightNormal, t.MutedForeground).TextAlign(ui.Center)
				if o.Title != "" {
					d.Margin(design.Sp(1), 0, 0, 0)
				}
			}
		})
		if content != nil {
			ui.Column(c).FillWidth().MaxWidth(384).AlignItems(ui.Center).Gap(design.Sp(4)).Children(content)
		}
	})
}

// AccordionItem is one section of an Accordion.
type AccordionItem struct {
	Title   string
	Open    *bool
	Content func()
}

// Accordion is coss's Accordion: sections divided by borders, each a 14/20
// medium trigger (16 DIPs above and below) with a chevron that turns over
// 200 ms, and a muted panel whose height animates over 200 ms.
func Accordion(c *ui.Context, items []AccordionItem) ui.Element {
	t := tok(c)
	return ui.Column(c).FillWidth().Children(func() {
		for i, it := range items {
			it := it
			item := ui.Column(c).Key(i).FillWidth()
			if i < len(items)-1 {
				item.BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(t.Border)
			}
			item.Children(func() {
				parts := ui.CollapsibleBase(c, it.Open)
				tr := parts.Trigger.FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Start).Gap(design.Sp(4)).PaddingY(design.Sp(4)).
					Radius(design.RadiusMD).FocusRing(false).Cursor(ui.CursorPointer)
				s := interact(tr, false)
				var sh shadowList
				if s.Has(StateFocusVisible) {
					sh.add(0, 0, 0, design.InputRingWidth, t.Ring)
				}
				finish(tr, &sh, nil)
				tr.Children(func() {
					text(c, it.Title, design.TextSM, design.WeightMedium, t.Foreground).Shrink(1)
					rot := b2f(*it.Open) * 180
					ic := icon(c, IconChevronDown, design.IconSM, t.Foreground, false).Margin(2, 0, 0, 0)
					if forced == 0 {
						rot = ic.AnimateWith("coss.chevron", rot, design.DurationPanel, design.EaseInOut)
					}
					ic.Rotate(rot)
				})
				parts.Panel(func() {
					ui.Column(c).FillWidth().Padding(0, 0, design.Sp(4), 0).TextColor(t.MutedForeground).FontSize(design.TextSM.Size).Children(it.Content)
				}).Clip()
			})
		}
	})
}

// Collapsible is coss's Collapsible: trigger shows content below it while
// *open, its height animating over 200 ms.
func Collapsible(c *ui.Context, open *bool, trigger func(open bool), content func()) ui.Element {
	return ui.Column(c).Children(func() {
		parts := ui.CollapsibleBase(c, open)
		parts.Trigger.Cursor(ui.CursorPointer).Children(func() { trigger(*open) })
		parts.Panel(content).Clip()
	})
}

// AvatarOptions configure an Avatar.
type AvatarOptions struct {
	// Size is the diameter, 32 by default; initials are 0.375 of it.
	Size  float32
	Image *ui.Bitmap
	Key   any
}

// Avatar is coss's Avatar: a round image, or the initials of name on a
// border-colored disc.
func Avatar(c *ui.Context, name string, o AvatarOptions) ui.Element {
	t := tok(c)
	size := o.Size
	if size == 0 {
		size = design.Sp(8)
	}
	withKey(c, o.Key)
	a := ui.Box(c).Size(size, size).Shrink(0).Radius(design.RadiusFull).Clip().Center().Background(t.Border).Label(name).Role(ui.RoleImage)
	a.Children(func() {
		if o.Image != nil {
			ui.Image(c, o.Image).Size(size, size).Fit(ui.Cover)
			return
		}
		text(c, initials(name), design.TextStyle{Size: size * 0.375, LineHeight: size * 0.375}, design.WeightMedium, t.Foreground)
	})
	return a
}

func initials(name string) string {
	var out []rune
	prev := ' '
	for _, r := range name {
		if prev == ' ' && r != ' ' && len(out) < 2 {
			if r >= 'a' && r <= 'z' {
				r -= 'a' - 'A'
			}
			out = append(out, r)
		}
		prev = r
	}
	return string(out)
}

// Breadcrumb is coss's Breadcrumb: muted links separated by chevrons, the
// last item the current page in foreground. It returns the index of the
// link clicked, -1 otherwise.
func Breadcrumb(c *ui.Context, items []string) int {
	t := tok(c)
	clicked := -1
	ui.Row(c).Wrap().AlignItems(ui.Center).Gap(design.Sp(2.5)).Role(ui.RoleList).Label("breadcrumb").Children(func() {
		for i, it := range items {
			if i > 0 {
				icon(c, IconChevronRight, design.IconSM, t.MutedForeground, false)
			}
			if i == len(items)-1 {
				text(c, it, design.TextSM, design.WeightNormal, t.Foreground)
				continue
			}
			c.Key(i)
			l := ui.ButtonBase(c).FocusRing(false).Cursor(ui.CursorPointer).Role(ui.RoleLink)
			s := interact(l, false)
			fg := t.MutedForeground
			if s.Has(StateHover) {
				fg = t.Foreground
			}
			var sh shadowList
			sh.ring(t, b2f(s.Has(StateFocusVisible)))
			finish(l, &sh, nil)
			l.Children(func() { text(c, it, design.TextSM, design.WeightNormal, fg) })
			if l.Clicked() {
				clicked = i
			}
		}
	})
	return clicked
}

// Pagination is coss's Pagination: Previous, page numbers (the current one
// outlined), ellipses and Next, 4 DIPs apart. Pages are 1-based; it moves
// *page when one is clicked and reports whether it did.
func Pagination(c *ui.Context, page *int, total int) bool {
	t := tok(c)
	changed := false
	set := func(p int) {
		if p >= 1 && p <= total && p != *page {
			*page, changed = p, true
		}
	}
	ui.Row(c).FillWidth().Justify(ui.Center).Role(ui.RoleGroup).Label("pagination").Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(1)).Children(func() {
			if Button(c, "Previous", ButtonOptions{Variant: ButtonGhost, Icon: IconChevronLeft, Disabled: *page <= 1, Key: "prev"}).Clicked() {
				set(*page - 1)
			}
			for _, p := range pageWindow(*page, total) {
				if p == 0 {
					ui.Row(c).MinWidth(28).Justify(ui.Center).Children(func() {
						icon(c, IconEllipsis, design.IconSM, t.Foreground, true)
					})
					continue
				}
				v := ButtonGhost
				if p == *page {
					v = ButtonOutline
				}
				label := strconv.Itoa(p)
				if Button(c, "", ButtonOptions{Variant: v, Size: ButtonIcon, Label: "Page " + label, Key: p, Content: func() {
					text(c, label, design.TextSM, design.WeightMedium, t.Foreground)
				}}).Clicked() {
					set(p)
				}
			}
			if Button(c, "Next", ButtonOptions{Variant: ButtonGhost, TrailingIcon: IconChevronRight, Disabled: *page >= total, Key: "next"}).Clicked() {
				set(*page + 1)
			}
		})
	})
	return changed
}

// pageWindow lists the pages to show around page, 0 for an ellipsis.
func pageWindow(page, total int) []int {
	if total <= 7 {
		out := make([]int, total)
		for i := range out {
			out[i] = i + 1
		}
		return out
	}
	out := []int{1}
	lo, hi := max(2, page-1), min(total-1, page+1)
	if lo > 2 {
		out = append(out, 0)
	}
	for p := lo; p <= hi; p++ {
		out = append(out, p)
	}
	if hi < total-1 {
		out = append(out, 0)
	}
	return append(out, total)
}

// Toolbar is coss's Toolbar: a card-colored strip (radius 14, 4 DIPs of
// padding, 8 between groups), one Tab stop with arrows between controls.
func Toolbar(c *ui.Context, label string, fn func()) ui.Element {
	t := tok(c)
	return ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(2)).Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).
		Padding(design.Sp(1)).Role(ui.RoleToolbar).Label(label).FocusGroup(ui.Horizontal).Children(fn)
}

// ToolbarGroup groups toolbar controls 4 DIPs apart.
func ToolbarGroup(c *ui.Context, fn func()) ui.Element {
	return ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(1)).Children(fn)
}

// ToolbarSeparator is a vertical toolbar divider (6 DIPs from the ends).
func ToolbarSeparator(c *ui.Context) ui.Element {
	return ui.Box(c).Width(1).AlignSelf(ui.Stretch).Margin(design.Sp(1.5), 0).Background(tok(c).Border).Shrink(0)
}

// TableVariant is the table's look.
type TableVariant uint8

const (
	TableDefault TableVariant = iota
	// TableCard puts the body in a rounded card.
	TableCard
)

// TableOptions configure a Table.
type TableOptions struct {
	Variant TableVariant
	// Widths fix column widths; 0 columns share the rest.
	Widths []float32
	// Selected highlights a row (-1 none).
	Selected int
	Footer   []string
	Caption  string
}

// Table is coss's Table: a 40 DIP header of muted medium labels, rows of
// 14/14 cells with 10 DIPs of padding divided by borders, hover and
// selected washes. It returns the index of the row clicked, -1 otherwise.
func Table(c *ui.Context, columns []string, rows [][]string, o TableOptions) int {
	t := tok(c)
	clicked := -1
	mix := func(a float32) ui.Color {
		if t.IsDark() {
			return t.Background.Mix(white, a)
		}
		return t.Background.Mix(black, a)
	}
	cell := func(i int, e ui.Element) ui.Element {
		if i < len(o.Widths) && o.Widths[i] > 0 {
			return e.Width(o.Widths[i]).Shrink(0)
		}
		return e.Grow(1).Basis(0).MinWidth(0)
	}
	leading := design.TextStyle{Size: design.TextSM.Size, LineHeight: design.TextSM.Size}
	ui.Column(c).FillWidth().Role(ui.RoleTable).Children(func() {
		ui.Row(c).FillWidth().Height(40).AlignItems(ui.Center).BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(t.Border).Role(ui.RoleRow).Children(func() {
			for i, col := range columns {
				cell(i, ui.Row(c).PaddingX(design.Sp(2.5)).Role(ui.RoleColumnHeader)).Children(func() {
					text(c, col, leading, design.WeightMedium, t.MutedForeground)
				})
			}
		})
		body := ui.Column(c).FillWidth()
		if o.Variant == TableCard {
			body.Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Clip()
			var sh shadowList
			sh.scale(design.ShadowXs, black, 0.05)
			finish(body, &sh, nil)
		}
		body.Children(func() {
			for r, row := range rows {
				c.Key(r)
				tr := ui.ButtonBase(c).FillWidth().Justify(ui.Start).FocusRing(false).Role(ui.RoleRow)
				if r < len(rows)-1 {
					tr.BorderWidth(0, 0, design.BorderWidth, 0).BorderColor(t.Border)
				}
				s := interact(tr, false)
				if o.Variant == TableDefault {
					switch {
					case r == o.Selected:
						tr.Background(mix(0.04))
					case s.Has(StateHover):
						tr.Background(mix(0.02))
					}
				}
				tr.Children(func() {
					for i, v := range row {
						cell(i, ui.Row(c).Padding(design.Sp(2.5)).Role(ui.RoleCell)).Children(func() {
							text(c, v, leading, design.WeightNormal, t.Foreground)
						})
					}
				})
				if tr.Clicked() {
					clicked = r
				}
			}
		})
		if len(o.Footer) > 0 {
			bg := t.Card.Mix(black, 0.02)
			if t.IsDark() {
				bg = t.Card.Mix(white, 0.02)
			}
			ui.Row(c).FillWidth().BorderWidth(design.BorderWidth, 0, 0, 0).BorderColor(t.Border).Background(bg).Children(func() {
				for i, v := range o.Footer {
					cell(i, ui.Row(c).Padding(design.Sp(3.5), design.Sp(2.5))).Children(func() {
						text(c, v, leading, design.WeightMedium, t.Foreground)
					})
				}
			})
		}
		if o.Caption != "" {
			para(c, o.Caption, design.TextSM, design.WeightNormal, t.MutedForeground).Margin(design.Sp(4), 0, 0, 0).TextAlign(ui.Center)
		}
	})
	return clicked
}

// ScrollAreaOptions configure a ScrollArea.
type ScrollAreaOptions struct {
	// State keeps the scroll position; nil keeps it on the element.
	State *ui.ScrollState
	// Horizontal scrolls sideways instead of down.
	Horizontal bool
	Key        any
}

type scrollMemo struct {
	st ui.ScrollState
}

const scrollAreaGutter float32 = 12

// ScrollArea is coss's ScrollArea: a scrollable container using native
// interactive scrollbars (draggable thumb & track paging) with a dedicated
// trailing gutter on vertical areas so thumbs never overlap trailing action
// controls such as row overflow menus. Size it with Height/Fill.
func ScrollArea(c *ui.Context, o ScrollAreaOptions, fn func()) ui.Element {
	withKey(c, o.Key)
	var sc ui.Element
	if o.Horizontal {
		sc = ui.ScrollHorizontal(c)
	} else {
		sc = ui.Scroll(c)
	}
	st := o.State
	if st == nil {
		memo := ui.Local(sc, "coss.scroll", func() scrollMemo { return scrollMemo{} })
		st = &memo.st
	}
	sc.TrackScroll(st)
	if forced.Has(StateHover) {
		sc.ScrollbarsVisible(true)
	}
	if o.Horizontal {
		sc.Children(fn)
	} else {
		gutter := float32(0)
		if st.MaxY > 0.5 || forced.Has(StateHover) {
			gutter = scrollAreaGutter
		}
		sc.Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Padding(0, gutter, 0, 0).Children(fn)
		})
	}
	return sc
}

// groupState is what Group tells the controls built in it: how many there
// are and which one comes next, so each rounds and borders only its outer
// side.
type groupState struct {
	n, next  int
	vertical bool
}

var currentGroup *groupState

// groupSlot reports a grouped control's place: whether it is first and last
// in its group, and whether the group is vertical. Outside a group it is
// both.
func groupSlot() (first, last, vertical, grouped bool) {
	g := currentGroup
	if g == nil {
		return true, true, false, false
	}
	i := g.next
	g.next++
	return i == 0, i == g.n-1, g.vertical, true
}

// Group is coss's Group: n controls (buttons, inputs, selects, GroupText)
// joined into one strip, their inner corners squared and inner borders
// collapsed.
func Group(c *ui.Context, n int, o Orientation, fn func()) ui.Element {
	g := ui.Row(c).Shrink(0).Role(ui.RoleGroup)
	if o == Vertical {
		g.Column()
	}
	saved := currentGroup
	currentGroup = &groupState{n: n, vertical: o == Vertical}
	g.Children(fn)
	currentGroup = saved
	return g
}

// joined squares e's inner corners and drops its inner borders as a Group
// member, keeping the full radius r outside.
func joined(e ui.Element, r float32, bc ui.Color) {
	first, last, vertical, grouped := groupSlot()
	if !grouped {
		return
	}
	rad := [4]float32{r, r, r, r}
	w := [4]float32{design.BorderWidth, design.BorderWidth, design.BorderWidth, design.BorderWidth}
	if vertical {
		if !first {
			rad[0], rad[1], w[0] = 0, 0, 0
		}
		if !last {
			rad[2], rad[3], w[2] = 0, 0, 0
		}
	} else {
		if !first {
			rad[0], rad[3], w[3] = 0, 0, 0
		}
		if !last {
			rad[1], rad[2], w[1] = 0, 0, 0
		}
	}
	e.Radius(rad[0], rad[1], rad[2], rad[3]).BorderWidth(w[0], w[1], w[2], w[3])
	if bc.A > 0 {
		e.BorderColor(bc)
	}
}

// GroupText is coss's GroupText: a muted label segment of a Group, in the
// text control frame.
func GroupText(c *ui.Context, s string) ui.Element {
	t := tok(c)
	bg := t.Muted
	if t.IsDark() {
		bg = t.Input.Alpha(0.64)
	}
	e := ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(2)).PaddingX(design.Sp(3)-1).Radius(design.RadiusLG).Border(design.BorderWidth, t.Input).Background(bg).Shrink(0)
	var sh shadowList
	sh.scale(design.ShadowXs, black, 0.05)
	d := deco{border: design.BorderWidth, radius: design.RadiusLG}
	d.highlight(t, 0.06, 0.06)
	finish(e, &sh, &d)
	joined(e, design.RadiusLG, t.Input)
	return e.Children(func() { text(c, s, design.TextSM, design.WeightNormal, t.MutedForeground) })
}

// InputGroupOptions configure an InputGroup.
type InputGroupOptions struct {
	InputOptions
	// Start and End build addons inside the frame: icons, muted text, Kbd
	// or small buttons.
	Start, End func()
}

// InputGroup is coss's InputGroup: an input with addons inside one text
// control frame (addons 11 DIPs from the edges, 8 from the text). It
// returns the text input.
func InputGroup(c *ui.Context, value *string, o InputGroupOptions) ui.Element {
	o.InputOptions.fromField()
	t := tok(c)
	h, padX := o.Size.inner()
	withKey(c, o.Key)
	frame := ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Role(ui.RoleGroup)
	var in ui.Element
	frame.Children(func() {
		addon := func(fn func(), start bool) {
			a := ui.Row(c).AlignItems(ui.Center).Gap(design.Sp(2)).Shrink(0).TextColor(t.MutedForeground)
			if start {
				a.Padding(0, 0, 0, padX)
			} else {
				a.Padding(0, padX, 0, 0)
			}
			a.Children(fn)
		}
		if o.Start != nil {
			addon(o.Start, true)
		}
		in = ui.TextInputBase(c, value).Grow(1).MinWidth(0).Height(h).FontSize(design.TextSM.Size).FixedLineHeight(h).
			TextColor(t.Foreground).FocusRing(false)
		left, right := padX, padX
		if o.Start != nil {
			left = design.Sp(2)
		}
		if o.End != nil {
			right = design.Sp(2)
		}
		in.Padding(0, right, 0, left)
		if o.Label != "" {
			in.Label(o.Label)
		}
		if *value == "" && o.Placeholder != "" {
			in.Placeholder(o.Placeholder)
		}
		if o.End != nil {
			addon(o.End, false)
		}
	})
	s := interactField(in, isDisabled(o.Disabled))
	if o.Invalid || o.Error != "" {
		s |= StateInvalid
	}
	controlLook(frame, t, s, design.RadiusLG)
	disabledLook(frame, s)
	return in
}

// AddonText is muted addon text for an InputGroup.
func AddonText(c *ui.Context, s string) ui.Element {
	return text(c, s, design.TextSM, design.WeightNormal, tok(c).MutedForeground)
}

// AddonIcon is an addon icon (16 DIPs, 80%).
func AddonIcon(c *ui.Context, i Icon) ui.Element {
	return icon(c, i, design.IconSM, tok(c).MutedForeground, false).MarginX(-2)
}
