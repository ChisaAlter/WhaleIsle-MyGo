package chrome

import (
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// GitLabel.module.css (installed ui-git/lib/client.js706) hides both labels in
// every density. GitActionsControl.js639–646 latches each hovered part until
// the pointer leaves the whole toolbar; keyboard focus and open menus reveal
// independently, so leaving a menu anchor must not hide its open label.
// The native input API exposes FocusVisible, but no pointer:coarse/touch kind.
// Do not infer that media query from viewport width or ordinary pointer focus.
type captionGitReveal struct{ branch, action bool }

func (r *captionGitReveal) observe(toolbar, branch, action bool) {
	if !toolbar {
		r.branch, r.action = false, false
		return
	}
	r.branch = r.branch || branch
	r.action = r.action || action
}

const captionGitLabelDuration = 200 * time.Millisecond

// captionGitLabel animates the source0fr/1fr, opacity and translateX together
// with the existing native animation clock. At rest it requests no more frames.
// The4-DIP prefix belongs inside the collapsed track, not the button's Gap.
func captionGitLabel(c *ui.Context, owner ui.Element, text, part string, shown bool, maxWidth float32) {
	target := float32(0)
	if shown {
		target = 1
	}
	p := owner.AnimateWith("git-label-reveal", target, captionGitLabelDuration, widgets.DesignEase)
	font, size := c.Theme().Font, theme.InterfaceSize(12)
	w, h := c.MeasureText(0, ui.Span{Text: text, Font: font, Size: size, Weight: 400})
	width := w + 4
	if maxWidth > 0 {
		width = min(width, maxWidth)
	}
	track := ui.Row(c.Key("git-label-" + part)).Label("Git " + part + " label").Width(width * p).Height(max(h, float32(18))).MinWidth(0).Shrink(0).Clip().Opacity(p)
	if p == 0 {
		track.Invisible()
	}
	track.Children(func() {
		ui.Text(c, text).Font(font).FontSize(size).FontWeight(400).FixedLineHeight(18).Width(width).MinWidth(0).SingleLine().Padding(0, 0, 0, 4).TextColor(c.Theme().Text).
			OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{X: -4 * (1 - p)} })
	})
}
