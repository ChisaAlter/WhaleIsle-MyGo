package chat

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func toolHeaderSummary(c *ui.Context, info ToolCardInfo, st ToolCardStyle, text string, size, height float32, ink ui.Color, running, active bool, exitLabel string) {
	t := design.For(c)
	font := theme.RenderPreferences().Chat.Family
	linked := info.summaryModel != nil && info.summaryModel.FilePath != "" && info.openFile != nil && !toolSummaryHasCue(info.Status) && exitLabel == ""
	webLinked := info.webHref != "" && info.openURL != nil && !toolSummaryHasCue(info.Status) && exitLabel == ""
	if linked || webLinked {
		// A nested base consumes its own click/Enter/Space. The parent retains
		// disclosure semantics, while this independently focusable action never
		// changes expansion and passes only the paired call's navigation data.
		button := ui.ButtonBase(c.Key("summary-file")).MinWidth(0).Shrink(1).Height(height).Padding(0).Label(text).
			Background(ui.Transparent).Cursor(ui.CursorPointer).Justify(ui.Start)
		if webLinked {
			button.Role(ui.RoleLink).FocusRing(false)
			focused := button.FocusVisible()
			button.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if focused {
					p.Stroke(ui.Rect{X: r.X - 3, Y: r.Y - 3, W: r.W + 6, H: r.H + 6}, t.Ring, design.RadiusXS+1, design.RingWidth)
				}
			})
		}
		space := false
		if webLinked {
			space = button.Shortcut(0, ui.KeySpace)
		}
		clicked := button.Clicked()
		if clicked && !(webLinked && space) {
			if webLinked {
				info.openURL(info.webHref)
			} else {
				target := info.summaryModel.FilePath
				if info.summaryModel.Line > 0 {
					target += ":" + strconv.Itoa(info.summaryModel.Line)
				}
				info.openFile(target)
			}
		}
		button.Children(func() {
			label := toolSummaryText(c, text, font, size, height, ink, running).MinWidth(0).Shrink(1)
			// CSS's dotted 1px underline sits 3px below the text baseline. Retain
			// native text/ellipsis and draw its decoration inside the same bounds.
			metrics := (ui.Font{Family: font, Size: size}).Metrics()
			baseline := (height-metrics.Ascent-metrics.Descent)/2 + metrics.Ascent
			label.DrawOver(func(p *ui.Painter, r ui.Rect) {
				y := r.Y + baseline + 3
				for x := r.X; x < r.X+r.W; x += 3 {
					p.Fill(ui.Rect{X: x, Y: y, W: min(float32(1), r.X+r.W-x), H: 1}, ink, .5)
				}
			})
		})
	} else {
		label := toolSummaryText(c, text, font, size, height, ink, running).MinWidth(0)
		if exitLabel == "" && info.summaryModel == nil {
			label.Grow(1)
		} else {
			label.Shrink(1)
		}
	}
	if exitLabel != "" {
		terminalExitBadge(c, exitLabel, st)
	}
	if toolSummaryHasCue(info.Status) {
		return
	}
	if info.image != nil {
		toolSummaryText(c, fmt.Sprintf("%s (%d)", i18n.T(i18n.KeyToolImageLabel), len(info.image.Images)), font, size, height, ink, running).Shrink(0).Margin(0, 0, 0, design.Sp(1))
	} else if info.contentKB != nil {
		toolSummaryText(c, fmt.Sprintf("%dKB", *info.contentKB), font, size, height, ink, running).Shrink(0).Margin(0, 0, 0, design.Sp(1))
	}
	if info.diffSummary != nil {
		added, removed := ink, ink
		if active {
			added, removed = t.SuccessForeground, t.DestructiveForeground
		}
		ui.Row(c).Shrink(0).Margin(0, 0, 0, design.Sp(2.5)).Height(height).AlignItems(ui.Center).Children(func() {
			// Match a single monospace inline text, including its real space.
			ui.RichText(c, ui.Span{Text: fmt.Sprintf("+%d", info.diffSummary.Added), Color: added}, ui.Span{Text: " "}, ui.Span{Text: fmt.Sprintf("-%d", info.diffSummary.Removed), Color: removed}).
				Font(theme.RenderPreferences().Code.Family).FontSize(11).FixedLineHeight(height).TextColor(ink).NoWrap().Shrink(0).Margin(design.Sp(0.125), 0, -design.Sp(0.125), 0)
		})
	}
}

func toolSummaryText(c *ui.Context, text, font string, size, height float32, ink ui.Color, running bool) ui.Element {
	if running {
		return chatTextShimmer(c, textShimmerOptions{Text: text, Active: true, Font: font, Size: size, Color: ink}).Height(height)
	}
	return ui.Text(c, text).Font(font).FontSize(size).FixedLineHeight(height).TextColor(ink).SingleLine()
}
