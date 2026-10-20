package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/theme"
)

// Terminal seat metrics from the Original TerminalBlock: one tinted container
// for the command and its output, 18 DIP lines, a horizontal scroll so long
// lines never widen the bubble, and the shared copy seat in the corner.
const (
	terminalPad        = 8
	terminalLineHeight = 18
	terminalMaxHeight  = 224
	terminalCopySize   = 20
)

// terminalBlockStyle carries the surface a terminal seat paints on.
type terminalBlockStyle struct {
	FG, Muted, Field ui.Color
	Size             float32
	// MaxHeight caps the vertical scroller; zero keeps the 224 DIP default.
	MaxHeight float32
}

// terminalBlock is the one container for shell-style tool evidence. command and
// output share the seat so a call reads as one terminal, and the copy action
// yields both.
func terminalBlock(c *ui.Context, st terminalBlockStyle, key, command, output string, onCopy func(string)) {
	command = strings.TrimRight(command, "\n")
	output = strings.TrimRight(output, "\n")
	if command == "" && output == "" {
		return
	}
	size := st.Size
	if size <= 0 {
		size = 11
	}
	maxHeight := st.MaxHeight
	if maxHeight <= 0 {
		maxHeight = terminalMaxHeight
	}
	font := theme.RenderPreferences().Code.Family
	payload := strings.TrimSpace(strings.Join([]string{strings.TrimSpace(command), output}, "\n"))
	toolSeat(c, toolSeatStyle{FG: st.FG, Muted: st.Muted, Field: st.Field, Size: size, MaxHeight: maxHeight},
		key, "Terminal output", payload, onCopy, func() {
			paintLine := func(text string, color ui.Color) {
				ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
					ui.Text(c, text).FontSize(size).Font(font).FixedLineHeight(terminalLineHeight).
						TextColor(color).NoWrap().Selectable()
				})
			}
			if command != "" {
				ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Start).Children(func() {
					ui.Text(c, "$").FontSize(size).Font(font).FixedLineHeight(terminalLineHeight).TextColor(st.Muted)
					ui.Box(c).Grow(1).MinWidth(0).Children(func() { paintLine(command, st.FG) })
				})
			}
			if output != "" {
				ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(maxHeight).Children(func() {
					ui.Column(c).FillWidth().MinWidth(0).Children(func() { paintLine(output, st.FG) })
				})
			}
		})
}
