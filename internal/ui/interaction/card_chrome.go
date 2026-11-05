package interaction

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// Each takeover owns header/body/footer; only its content body scrolls.
const (
	interactionCardRadius = 20
	interactionCardIndent = 16
)

func interactionCardSeat(c *ui.Context, key string, warn bool, tint ui.Color) ui.Element {
	card := ui.Column(c.Key(key)).FillWidth().MinWidth(0).MinHeight(0).Radius(interactionCardRadius).Clip().Gap(0).Background(c.Theme().Surface)
	if warn {
		card.Border(1, c.Theme().Warning.Alpha(.45))
		if residentComposerBounds.W > 0 {
			card.FillWidth().MaxWidth(residentComposerBounds.W).AlignSelf(ui.Center)
		}
		if residentComposerBounds.H > 0 {
			card.Height(residentComposerBounds.H)
		}
	} else {
		_, height := c.Size()
		card.MaxHeight(min(float32(520), height*.6)).Padding(0, 0, 10, 0)
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
	ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
		ui.Text(c, command).Font(theme.RenderPreferences().Code.Family).FontSize(13).FixedLineHeight(20).TextColor(t.TextMuted).Selectable().MinWidth(0)
		if onCopy != nil && ui.Button(c, i18n.T(i18n.KeyChatToolCopy)).FontSize(11).Height(24).BorderWidth(0).Background(ui.Transparent).Clicked() {
			onCopy(command)
		}
	})
}
