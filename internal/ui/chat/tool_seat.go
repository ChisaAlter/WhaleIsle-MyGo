package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// Every tool family renders its evidence in the same seat: one tinted container,
// an 8 DIP padding, a 10 DIP copy action in the corner and the same 18 DIP line
// rhythm. Only the body differs — terminal, search hits, web cards, todo rows.
const (
	toolSeatRadius   = 8
	toolSeatPad      = 8
	toolSeatLine     = 18
	toolSeatCopySize = 20
	toolSeatMaxBody  = 224
)

// toolSeatStyle carries the surface a tool seat paints on.
type toolSeatStyle struct {
	FG, Muted, Field ui.Color
	// Accent colors titles that open a link; zero keeps the body color.
	Accent ui.Color
	Size   float32
	// MaxHeight caps an inner scroller; zero keeps the 224 DIP default.
	MaxHeight float32
}

func (s toolSeatStyle) size() float32 {
	if s.Size > 0 {
		return s.Size
	}
	return 11
}

func (s toolSeatStyle) maxHeight() float32 {
	if s.MaxHeight > 0 {
		return s.MaxHeight
	}
	return toolSeatMaxBody
}

// toolSeat paints the shared frame around a family-specific body. payload is what
// the copy action yields; an empty payload renders no copy seat.
func toolSeat(c *ui.Context, st toolSeatStyle, key, label, payload string, onCopy func(string), body func()) {
	seat := ui.Column(c).Key("tool-seat:"+key).Label(label).FillWidth().MinWidth(0).Selectable().
		Background(st.Field).Radius(toolSeatRadius).Padding(toolSeatPad, toolSeatPad, toolSeatPad, toolSeatPad).Gap(2)
	copied := ui.Local(seat, "tool-seat-copied", func() string { return "" })
	seat.Children(func() {
		if onCopy != nil && payload != "" {
			ui.Row(c).FillWidth().MinWidth(0).Unselectable().Justify(ui.End).Children(func() {
				text := i18n.T(i18n.KeyChatToolCopy)
				if *copied == payload {
					text = i18n.T(i18n.KeyMarkdownCopied)
				}
				button := ui.Button(c, text).Label(text).Tooltip(text).FontSize(10).Height(toolSeatCopySize).
					Padding(0, 8, 0, 8).Radius(6).Border(0.5, st.Muted.Alpha(.35)).
					Background(ui.Transparent).TextColor(st.Muted)
				if button.Clicked() {
					if *copied == payload {
						*copied = ""
					} else {
						*copied = payload
					}
					onCopy(payload)
				}
			})
		}
		body()
	})
}
