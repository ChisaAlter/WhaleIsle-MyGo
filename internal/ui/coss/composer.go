package coss

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// ComposerSuggestion is one row of the composer's mention/slash popup.
type ComposerSuggestion struct {
	ID, Label, Detail string
	Icon              Icon
}

// ComposerOptions configure a Composer.
type ComposerOptions struct {
	Placeholder string
	Disabled    bool
	Invalid     bool
	Label       string
	// MaxRows is how far the input grows before it scrolls (4; 8 on
	// windows at least 768 wide, as coss's md:).
	MaxRows int
	// Start and End build the persistent action groups below the input
	// (attach, model pickers…; End usually ends with the submit button,
	// which Submit adds when End is nil).
	Start, End func()
	// SubmitLabel names the default submit button.
	SubmitLabel string
	// Triggers open the suggestion popup: '@' for mentions, '/' for
	// commands. Suggest returns the rows for a trigger and the text typed
	// after it; nil Suggest disables the popup.
	Triggers string
	Suggest  func(trigger byte, query string) []ComposerSuggestion
	Key      any
}

// ComposerEvents report what happened in a composer this frame.
type ComposerEvents struct {
	// Submit is set by Enter (without Shift) or the submit button while the
	// text is not empty; the caller sends *value and clears it.
	Submit bool
	// Picked is the suggestion chosen; the composer replaced the trigger
	// and query with its label.
	Picked *ComposerSuggestion
	// Input is the text area, for focus and Changed.
	Input ui.Element
}

type composerState struct {
	highlight int
	// enter and move are keys the text area handed over since the last
	// build (OnShortcut runs after construction).
	enter bool
	move  int
}

// Composer is coss's Composer (stacked layout): a 14 DIP-radius text
// control frame holding a growing text area (one line at least, MaxRows at
// most, then scrolling) above persistent action groups, with a mention and
// slash-command popup above it.
func Composer(c *ui.Context, value *string, o ComposerOptions) ComposerEvents {
	io := InputOptions{Label: o.Label, Invalid: o.Invalid, Disabled: o.Disabled}
	io.fromField()
	t := tok(c)
	var ev ComposerEvents
	withKey(c, o.Key)
	frame := ui.Column(c).FillWidth().MinWidth(0).Padding(design.Sp(2), 0, design.Sp(2)-1, 0).Cursor(ui.CursorText)
	st := ui.Local(frame, "coss.composer", func() composerState { return composerState{} })
	maxRows := o.MaxRows
	if maxRows <= 0 {
		maxRows = 4
		if w, _ := c.Size(); w >= 768 {
			maxRows = 8
		}
	}
	disabled := isDisabled(io.Disabled)
	trig, query, at := activeTrigger(*value, o.Triggers)
	var rows []ComposerSuggestion
	if o.Suggest != nil && trig != 0 {
		rows = o.Suggest(trig, query)
	}
	if st.highlight >= len(rows) {
		st.highlight = 0
	}
	pick := func(i int) {
		r := rows[i]
		*value = (*value)[:at] + string(trig) + r.Label + " "
		ev.Picked = &r
		st.highlight = 0
	}
	frame.Children(func() {
		in := ui.TextAreaBase(c, value).FillWidth().Padding(design.Sp(1), design.Sp(3)-1).
			FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).TextColor(t.Foreground).FocusRing(false).
			Lines(1, maxRows)
		if io.Label != "" {
			in.Label(io.Label)
		}
		if *value == "" && o.Placeholder != "" {
			in.Placeholder(o.Placeholder)
			placeholder(c, t, o.Placeholder, design.Sp(3)-1, design.Sp(2)+design.Sp(1), design.TextSM.LineHeight, design.TextSM)
		}
		ev.Input = in
		if st.move != 0 && len(rows) > 0 {
			st.highlight = (st.highlight + st.move + len(rows)) % len(rows)
		}
		st.move = 0
		if st.enter {
			st.enter = false
			if len(rows) > 0 {
				pick(st.highlight)
			} else if strings.TrimSpace(*value) != "" && !disabled {
				ev.Submit = true
			}
		}
		// The text area would take Enter as a newline: while it has the focus
		// (and no input method is composing), Enter submits or picks and the
		// arrows and Tab drive the suggestions; Shift+Enter stays a newline.
		if in.Focused() && !in.Composing() {
			hasRows := len(rows) > 0
			c.CaptureKeyDown(func(e ui.InputEvent) bool {
				if e.Mods != 0 {
					return false
				}
				switch {
				case e.Key == ui.KeyEnter, hasRows && e.Key == ui.KeyTab:
					st.enter = true
				case hasRows && e.Key == ui.KeyDown:
					st.move = 1
				case hasRows && e.Key == ui.KeyUp:
					st.move = -1
				default:
					return false
				}
				return true
			})
		}
		if len(rows) > 0 {
			composerPopup(c, t, frame, rows, st, pick)
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(design.Sp(2)).Margin(0, design.Sp(2)-1, 0, design.Sp(2)-1).Children(func() {
			ui.Row(c).MinWidth(0).Wrap().AlignItems(ui.Center).Gap(design.Sp(1)).Children(func() {
				if o.Start != nil {
					o.Start()
				}
			})
			ui.Row(c).MinWidth(0).Wrap().AlignItems(ui.Center).Justify(ui.End).Gap(design.Sp(1)).Children(func() {
				if o.End != nil {
					o.End()
					return
				}
				empty := strings.TrimSpace(*value) == ""
				if Button(c, "", ButtonOptions{Size: ButtonIconSM, Icon: IconArrowUp, Label: firstNonEmpty(o.SubmitLabel, "Send"), Disabled: empty || disabled}).Radius(design.RadiusFull).Clicked() {
					ev.Submit = true
				}
			})
		})
	})
	s := interactField(ev.Input, disabled)
	if io.Invalid {
		s |= StateInvalid
	}
	controlLook(frame, t, s, design.RadiusXL)
	disabledLook(frame, s)
	return ev
}

// activeTrigger finds a trigger character starting the word at the end of
// s (at the start or after a space), the query after it and its index.
func activeTrigger(s, triggers string) (byte, string, int) {
	if triggers == "" || s == "" {
		return 0, "", 0
	}
	i := strings.LastIndexAny(s, " \n\t")
	word := s[i+1:]
	if word == "" || !strings.ContainsRune(triggers, rune(word[0])) {
		return 0, "", 0
	}
	return word[0], word[1:], i + 1
}

// composerPopup is coss's ComposerPopup: a popover above the composer, as
// wide as it, listing suggestions (rows 28 high, accent while highlighted).
func composerPopup(c *ui.Context, t *design.Tokens, anchor ui.Element, rows []ComposerSuggestion, st *composerState, pick func(int)) {
	open := true
	ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
		panel.AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 4, 0).Width(anchor.Bounds().W)
		composerList(c, t, panel, rows, st, pick)
	})
}

// ComposerPanel draws the suggestion popup in place (the gallery's).
func ComposerPanel(c *ui.Context, rows []ComposerSuggestion, highlight int) ui.Element {
	t := tok(c)
	p := ui.Column(c)
	st := &composerState{highlight: highlight}
	composerList(c, t, p, rows, st, func(int) {})
	return p
}

func composerList(c *ui.Context, t *design.Tokens, panel ui.Element, rows []ComposerSuggestion, st *composerState, pick func(int)) {
	panelLook(panel, t, design.RadiusLG, design.ShadowLg)
	motion(panel, "composer/composer-popup", 0, 0)
	panel.Role(ui.RoleList).Label("Suggestions").Children(func() {
		ui.Column(c).FillWidth().Padding(design.Sp(1)).Children(func() {
			for i, r := range rows {
				c.Key(r.ID)
				item := ui.ButtonBase(c).FillWidth().MinHeight(28).Radius(design.RadiusSM).Gap(design.Sp(2)).Padding(design.Sp(1), design.Sp(2)).
					Justify(ui.Start).FocusRing(false).Role(ui.RoleListItem)
				if item.Hovered() && forced == 0 {
					st.highlight = i
				}
				fg := t.Foreground
				if i == st.highlight {
					item.Background(t.Accent)
					fg = t.AccentForeground
				}
				item.Children(func() {
					if r.Icon != IconNone {
						icon(c, r.Icon, design.IconSM, fg, false)
					}
					text(c, r.Label, design.TextSM, design.WeightNormal, fg)
					if r.Detail != "" {
						ui.Spacer(c)
						text(c, r.Detail, design.TextXS, design.WeightNormal, t.MutedForeground)
					}
				})
				if item.Clicked() {
					pick(i)
				}
			}
		})
	})
}
