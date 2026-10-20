package interaction

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// The interaction cards share one chrome: a 16 DIP radius panel with a hairline
// border, an optional 3 DIP state bar on its leading edge and a quiet header row
// that names what the card is waiting for (Original ApprovalPanel.module.css).
const (
	interactionCardRadius = 16
	interactionCardIndent = 14
)

// interactionCardSeat is the panel every pending card sits in. warn paints the
// leading state bar in the warning token; the caller owns the tint.
func interactionCardSeat(c *ui.Context, key string, warn bool, tint ui.Color) ui.Element {
	border := c.Theme().Border
	if warn {
		border = c.Theme().Warning.Alpha(.45)
	}
	card := ui.Column(c).Key(key).FillWidth().MinWidth(0).
		Background(tint).Border(0.5, border).Radius(interactionCardRadius).
		Padding(12, interactionCardIndent, 12, interactionCardIndent).Gap(8)
	if warn {
		bar := c.Theme().Warning
		card.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + 6, W: 3, H: r.H - 12}, bar, 2)
		})
	}
	return card
}

// interactionCardHeader is the "waiting for you" line: a state dot, the quiet
// label and an optional trailing detail (a call id), never a second title.
func interactionCardHeader(c *ui.Context, state widgets.State, label, trailing string, muted ui.Color) {
	ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
		widgets.StateDot(c, state)
		ui.Text(c, label).FontSize(11).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
		if trailing != "" {
			ui.Text(c, trailing).FontSize(10).TextColor(muted.Alpha(.9)).SingleLine()
		}
	})
}

// interactionPrimaryButton is the card's confirming action, on the harness
// primary recipe (sm: 28 DIP, 12px text).
func interactionPrimaryButton(c *ui.Context, colors theme.Colors, label string, disabled bool) ui.Element {
	return widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: disabled})
}

// interactionGhostButton is the card's quiet action: no face until hovered.
func interactionGhostButton(c *ui.Context, colors theme.Colors, label string, disabled bool) ui.Element {
	return widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: disabled})
}

// interactionOutlineButton is the card's refusing action: harness outline stroke
// over the card surface. The primitive has no danger-text variant, so the
// refusal reads through position and the label rather than a red hover.
func interactionOutlineButton(c *ui.Context, colors theme.Colors, label string, disabled bool) ui.Element {
	return widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: disabled})
}

// approvalCommand extracts the command an approval is really about: a shell
// tool's `command`, a script, or a bare string argument. Everything else stays
// in the raw-arguments disclosure.
func approvalCommand(args string) string {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return ""
	}
	var payload struct {
		Command string `json:"command"`
		Cmd     string `json:"cmd"`
		Script  string `json:"script"`
	}
	if json.Unmarshal([]byte(trimmed), &payload) == nil {
		for _, candidate := range []string{payload.Command, payload.Cmd, payload.Script} {
			if text := strings.TrimSpace(candidate); text != "" {
				return text
			}
		}
	}
	var plain string
	if json.Unmarshal([]byte(trimmed), &plain) == nil {
		plain = strings.TrimSpace(plain)
		if plain != "" && !strings.HasPrefix(plain, "{") && !strings.HasPrefix(plain, "[") {
			return plain
		}
	}
	return ""
}

// commandBlock paints the monospace command seat: the exact command, scrollable
// when long, never reflowed into prose, with a quiet copy action beneath it.
func commandBlock(c *ui.Context, t *ui.Theme, command string, onCopy func(string)) {
	font := theme.RenderPreferences().Code.Family
	ui.Column(c).FillWidth().MinWidth(0).Radius(10).Background(t.Surface).Padding(8, 10, 8, 10).Gap(4).
		Children(func() {
			ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(140).Children(func() {
				ui.Text(c, command).FontSize(11).FixedLineHeight(17).Font(font).TextColor(t.Text).MinWidth(0)
			})
			if onCopy == nil {
				return
			}
			ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
				seat := ui.Button(c, i18n.T(i18n.KeyChatToolCopy)).FontSize(10).Height(20).Padding(0, 8, 0, 8).Radius(6).
					Border(0.5, t.Border).Background(ui.Transparent).TextColor(t.TextMuted)
				if seat.Clicked() {
					onCopy(command)
				}
			})
		})
}
