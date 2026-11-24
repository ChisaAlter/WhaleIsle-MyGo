// Package pagekit holds the page-level pieces the tool pages (Plugins,
// Plugin Market, Scheduled Tasks, Usage) share on top of internal/ui/coss:
// the page frame and header, the coss type ramp for running text, inline
// failure feedback that never shows a raw error (docs/error-copy.md), loading
// skeletons, and the memo-region versions those pages key their lists by.
//
// Everything here reads design.For(c); nothing does I/O, starts goroutines
// or timers.
package pagekit

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
)

// Text styles of the pages' running text, the coss type ramp.

// Title is a page title: 20/28 semibold, heading level 1.
func Title(c *ui.Context, s string) ui.Element {
	t := design.For(c)
	return ui.Text(c, s).FontSize(design.TextXL.Size).FixedLineHeight(design.TextXL.LineHeight).FontWeight(design.WeightSemibold).
		TextColor(t.Foreground).Role(ui.RoleHeading).Level(1)
}

// SectionTitle is a section heading: 14/20 semibold, level 2.
func SectionTitle(c *ui.Context, s string) ui.Element {
	t := design.For(c)
	return ui.Text(c, s).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).FontWeight(design.WeightSemibold).
		TextColor(t.Foreground).Role(ui.RoleHeading).Level(2)
}

// Strong is 14/20 medium foreground text: a row's name.
func Strong(c *ui.Context, s string) ui.Element {
	t := design.For(c)
	return ui.Text(c, s).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).FontWeight(design.WeightMedium).TextColor(t.Foreground)
}

// Body is 14/20 foreground text.
func Body(c *ui.Context, s string) ui.Element {
	t := design.For(c)
	return ui.Text(c, s).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).TextColor(t.Foreground)
}

// Muted is 14/20 muted-foreground text: descriptions.
func Muted(c *ui.Context, s string) ui.Element {
	t := design.For(c)
	return ui.Text(c, s).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).TextColor(t.MutedForeground)
}

// Caption is 12/16 muted-foreground text: metadata lines.
func Caption(c *ui.Context, s string) ui.Element {
	t := design.For(c)
	return ui.Text(c, s).FontSize(design.TextXS.Size).FixedLineHeight(design.TextXS.LineHeight).TextColor(t.MutedForeground)
}

// Code is 12/16 monospace muted text: package names, specs, paths.
func Code(c *ui.Context, s string) ui.Element {
	return Caption(c, s).Font("monospace")
}

// Danger is 14/20 destructive-foreground text, for a short inline problem
// beside the control it concerns. s must already be a sentence.
func Danger(c *ui.Context, s string) ui.Element {
	t := design.For(c)
	return ui.Text(c, s).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).TextColor(t.DestructiveForeground)
}

// PageOptions configure a Page.
type PageOptions struct {
	// MaxWidth caps the content column (default 960).
	MaxWidth float32
	// Embedded drops the scroll container and insets: the host (a
	// Settings section) owns them.
	Embedded bool
	// Scroll keeps the page's scroll position across frames.
	Scroll *ui.ScrollState
	Key    any
}

// Page is a tool page's frame: a vertical scroll filling the space it is
// given, with responsive side insets, holding one centered column whose
// sections stand 24 DIPs apart. It returns the content column.
func Page(c *ui.Context, o PageOptions, fn func()) ui.Element {
	maxW := o.MaxWidth
	if maxW == 0 {
		maxW = 960
	}
	if o.Key != nil {
		c.Key(o.Key)
	}
	var col ui.Element
	column := func() {
		col = ui.Column(c).FillWidth().MaxWidth(maxW).MinWidth(0).Gap(design.Sp(6)).Children(fn)
	}
	if o.Embedded {
		ui.Column(c).FillWidth().MinWidth(0).Children(column)
		return col
	}
	w, _ := c.Size()
	inset := max(design.Sp(6), min(design.Sp(12), w*0.04))
	sc := ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).MinWidth(0).Padding(design.Sp(7), inset, design.Sp(12), inset).AlignItems(ui.Center)
	if o.Scroll != nil {
		sc.TrackScroll(o.Scroll)
	}
	sc.Children(column)
	return col
}

// Header is a page header: the title with a muted description under it,
// and the page's actions at the right, wrapping under on narrow windows.
func Header(c *ui.Context, title, description string, actions func()) ui.Element {
	return ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(4)).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).Basis(0).MinWidth(240).Gap(design.Sp(1)).Children(func() {
			Title(c, title)
			if description != "" {
				Muted(c, description).FillWidth()
			}
		})
		if actions != nil {
			ui.Row(c).Shrink(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(actions)
		}
	})
}

// Section is a titled group: the heading (and an action at its right),
// then the content 12 DIPs below.
func Section(c *ui.Context, title string, action func(), fn func()) ui.Element {
	return ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(3)).Children(func() {
		if title != "" || action != nil {
			ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				SectionTitle(c, title).Grow(1).MinWidth(0)
				if action != nil {
					ui.Row(c).Shrink(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(action)
				}
			})
		}
		fn()
	})
}

// Back is the ghost "back" button of a detail view.
func Back(c *ui.Context, label string) bool {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Icon: coss.IconArrowLeft}).Margin(0, 0, 0, -design.Sp(2)).Clicked()
}

// Fault is the inline error Alert of a failure the page kept only as text:
// the localized sentence errcopy gives for raw (never raw itself), under
// title when one is given, with a retry action when retry is set. It draws
// nothing for an empty raw.
func Fault(c *ui.Context, title, raw, retryLabel string, retry func()) {
	sentence := errcopy.SentenceText(raw)
	if sentence == "" {
		return
	}
	FaultSentence(c, title, sentence, retryLabel, retry)
}

// FaultSentence is Fault for a sentence already made safe for the view
// (state.Fault.SentenceText, authored copy).
func FaultSentence(c *ui.Context, title, sentence, retryLabel string, retry func()) {
	if sentence == "" {
		return
	}
	o := coss.AlertOptions{Variant: coss.AlertError}
	if retry != nil {
		o.Action = func() {
			if coss.Button(c, retryLabel, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
				retry()
			}
		}
	}
	if title == "" {
		title, sentence = sentence, ""
	}
	coss.Alert(c, title, sentence, o)
}

// Note is an inline informational Alert (variant info, warning or
// success): the app's feedback in place of a toast.
func Note(c *ui.Context, variant coss.AlertVariant, title, description string, action func()) {
	if title == "" && description == "" {
		return
	}
	coss.Alert(c, title, description, coss.AlertOptions{Variant: variant, Action: action})
}

// SkeletonRows is a loading list: n rows of a 40 DIP tile and two text
// bars, announced by label.
func SkeletonRows(c *ui.Context, n int, label string) {
	ui.Column(c).FillWidth().Gap(design.Sp(3)).Role(ui.RoleStatus).Label(label).Children(func() {
		for i := 0; i < n; i++ {
			ui.Row(c).FillWidth().Gap(design.Sp(3)).AlignItems(ui.Center).Children(func() {
				coss.Skeleton(c).Size(40, 40).Radius(design.RadiusLG)
				ui.Column(c).Grow(1).Gap(design.Sp(2)).Children(func() {
					coss.Skeleton(c).WidthPercent(36).Height(design.Sp(3.5))
					coss.Skeleton(c).WidthPercent(68).Height(design.Sp(3))
				})
			})
		}
	})
}
