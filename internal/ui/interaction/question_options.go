package interaction

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The question options are selectable cards, not bare checkboxes: a 20 DIP
// index badge, the label and its description, and an accent border plus tint
// once chosen (Original UserQuestionsPanel option rows).
const questionOptionBadge = 20

var questionOptionCheckIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round">
    <path d="m3.4 8.4 3 3 6.2-6.8"/>
  </svg>`))

// questionPager icons turn the wizard's page buttons into the Original pager
// chevrons while keeping a spoken label.
var (
	questionPagerPrev = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M10 3.4 5.4 8 10 12.6"/></svg>`))
	questionPagerNext = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M6 3.4 10.6 8 6 12.6"/></svg>`))
	// Header affordances share the same 16px grid as the pager chevrons:
	// IconChevronDown/UpOutlineRegular for minimize, IconCloseOutlineRegular
	// for the single dismiss seat (QuestionComposer.tsx:511-528).
	questionChevronDown = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M3.4 6 8 10.6 12.6 6"/></svg>`))
	questionChevronUp   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M3.4 10 8 5.4 12.6 10"/></svg>`))
	questionCloseIcon   = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M4.4 4.4l7.2 7.2M11.6 4.4l-7.2 7.2"/></svg>`))
	// Native-only draft recap seat: a plain list glyph, no Original counterpart.
	questionReviewIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M3 4.6h10M3 8h10M3 11.4h6.4"/></svg>`))
)

// questionIconButton is the Original's .iconButton recipe
// (QuestionComposer.module.css:200-225): a 24 DIP round seat, borderless and
// transparent until hover lifts the interactive-bg fill and the glyph reads
// label-primary. The module declares no transition on it, so none is added.
func questionIconButton(c *ui.Context, icon *ui.SVG, label, tooltip string, enabled bool, colors theme.Colors) ui.Element {
	t := c.Theme()
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant:  coss.ButtonGhost,
		Size:     coss.ButtonIconXS,
		Disabled: !enabled,
		Label:    label,
		Content: func() {
			ui.Icon(c, icon).Width(14).Height(14)
		},
	}).Tooltip(tooltip).Radius(design.RadiusFull).Padding(0).TextColor(t.TextMuted)
	if !enabled {
		button.Opacity(.4)
	} else if button.Pressed() || button.Hovered() {
		button.Background(theme.Color(colors.Hover)).TextColor(t.Text)
	}
	return button
}

// questionOptionCard paints one option and returns its row so the caller can
// attach the wizard's own keys. The whole card is the mouse target; multi
// select answers keep check-box semantics, single select answers radio ones.
func questionOptionCard(c *ui.Context, key string, index int, label, description string, checked, multi, disabled, recommended bool, onToggle func()) ui.Element {
	t := c.Theme()
	border, background := ui.Transparent, ui.Transparent
	if checked && !multi {
		border, background = t.Border, t.TextMuted.Alpha(.10)
	}
	card := ui.Row(c.Key(key)).Label("Option "+label).FillWidth().MinWidth(0).MinHeight(40).Gap(8).AlignItems(ui.Start).Padding(8, 12, 8, 8).Radius(12).Border(1, border).Background(background).Selected(checked).Focusable().Disabled(disabled)
	if multi {
		card.Role(ui.RoleCheckBox)
	} else {
		card.Role(ui.RoleRadio)
	}
	if card.Hovered() && !disabled {
		card.Background(t.TextMuted.Alpha(.10))
	}
	card.Children(func() {
		badge := ui.Box(c).Size(20, 20).Margin(2, 0, 0, 0).Shrink(0).Center()
		badge.Children(func() {
			if multi {
				box := ui.Box(c).Size(14, 14).Radius(4).Border(.5, t.Border).Center()
				if checked {
					box.Background(t.Text).Border(.5, t.Text)
				}
				box.Children(func() {
					if checked {
						ui.Icon(c, questionOptionCheckIcon).Size(12, 12).TextColor(t.Background)
					}
				})
			} else {
				badge.Background(t.Surface).Radius(4)
				ui.Text(c, strconv.Itoa(index+1)).FontSize(12).FixedLineHeight(18).FontWeight(500).TextColor(t.TextMuted)
			}
		})
		ui.Row(c).Grow(1).MinWidth(0).Wrap().GapX(6).GapY(2).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(14).FixedLineHeight(24).FontWeight(500).TextColor(t.Text).MinWidth(0).Shrink(1)
			if recommended {
				ui.Text(c, i18n.T(i18n.KeyChatQuestionRecommended)).FontSize(11).FixedLineHeight(18).FontWeight(600).TextColor(t.Accent).Padding(0, 4).Radius(4).Background(t.Accent.Alpha(.12)).Shrink(0)
			}
			if description != "" {
				ui.Text(c, description).FontSize(14).FixedLineHeight(24).TextColor(t.TextMuted).MinWidth(0).Shrink(1)
			}
		})
	})
	if !disabled && onToggle != nil && card.Clicked() {
		onToggle()
	}
	return card
}
