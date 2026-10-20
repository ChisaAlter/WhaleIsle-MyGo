package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Terminal seat metrics from the Original TerminalBlock: one tinted container
// for the command and its output, 8 DIP padding, 18 DIP lines, a horizontal
// scroll so long lines never widen the bubble, and a copy action in the corner.
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
	font := theme.RenderPreferences().Code.Family
	maxHeight := st.MaxHeight
	if maxHeight <= 0 {
		maxHeight = terminalMaxHeight
	}
	payload := strings.TrimSpace(strings.Join([]string{strings.TrimSpace(command), output}, "\n"))
	seat := ui.Column(c).Key("terminal:"+key).Label("Terminal output").FillWidth().MinWidth(0).
		Background(st.Field).Radius(8).Padding(terminalPad, terminalPad, terminalPad, terminalPad).Gap(2)
	copied := ui.Local(seat, "terminal-copied", func() string { return "" })
	seat.Children(func() {
		if onCopy != nil && payload != "" {
			ui.Row(c).FillWidth().MinWidth(0).Justify(ui.End).Children(func() {
				label := i18n.T(i18n.KeyChatToolCopy)
				if *copied == payload {
					label = i18n.T(i18n.KeyMarkdownCopied)
				}
				button := ui.Button(c, label).Label(label).Tooltip(label).FontSize(10).Height(terminalCopySize).
					Padding(0, 8, 0, 8).Radius(6).Border(0.5, st.Muted.Alpha(.35)).
					Background(ui.Transparent).TextColor(st.Muted)
				if button.Clicked() {
					if *copied == payload {
						*copied = ""
					} else {
						*copied = payload
					}
					if onCopy != nil {
						onCopy(payload)
					} else {
						c.WriteClipboard(payload)
					}
				}
			})
		}
		paintTerminalLine := func(text string, color ui.Color) {
			ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
				ui.Text(c, text).FontSize(size).Font(font).FixedLineHeight(terminalLineHeight).
					TextColor(color).NoWrap().Selectable()
			})
		}
		if command != "" {
			ui.Row(c).FillWidth().MinWidth(0).Gap(6).AlignItems(ui.Start).Children(func() {
				ui.Text(c, "$").FontSize(size).Font(font).FixedLineHeight(terminalLineHeight).TextColor(st.Muted)
				ui.Box(c).Grow(1).MinWidth(0).Children(func() { paintTerminalLine(command, st.FG) })
			})
		}
		if output != "" {
			ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(maxHeight).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Children(func() { paintTerminalLine(output, st.FG) })
			})
		}
	})
}
