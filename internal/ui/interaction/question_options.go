package interaction

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// The question options are selectable cards, not bare checkboxes: a 20 DIP
// index badge, the label and its description, and an accent border plus tint
// once chosen (Original UserQuestionsPanel option rows).
const questionOptionBadge = 20

var questionOptionCheckIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="m3.4 8.4 3 3 6.2-6.8"/>
  </svg>`))

// questionPager icons turn the wizard's page buttons into the Original pager
// chevrons while keeping a spoken label.
var (
	questionPagerPrev = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M10 3.4 5.4 8 10 12.6"/></svg>`))
	questionPagerNext = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M6 3.4 10.6 8 6 12.6"/></svg>`))
)

// questionPagerButton is one 28 DIP chevron seat of the wizard pager.
func questionPagerButton(c *ui.Context, icon *ui.SVG, label string, enabled bool) ui.Element {
	t := c.Theme()
	button := ui.Button(c, "").Label(label).Tooltip(label).Width(28).Height(28).Radius(8).
		Background(ui.Transparent).Border(0.5, t.Border).Padding(0, 0, 0, 0)
	button.Children(func() {
		ui.Icon(c, icon).Width(14).Height(14).TextColor(t.TextMuted)
	})
	if !enabled {
		button.Disabled(true).Opacity(.4)
	}
	return button
}

// questionOptionCard paints one option and returns its row so the caller can
// attach the wizard's own keys. The whole card is the mouse target; multi
// select answers keep check-box semantics, single select answers radio ones.
func questionOptionCard(c *ui.Context, key string, index int, label, description string, checked, multi, disabled, recommended bool, onToggle func()) ui.Element {
	t := c.Theme()
	border, background := t.Border, ui.Transparent
	if checked {
		border, background = t.Accent.Alpha(.70), t.Accent.Alpha(.08)
	}
	card := ui.Row(c).Key(key).Label("Option " + label).FillWidth().MinWidth(0).
		Gap(10).AlignItems(ui.Center).Padding(8, 10, 8, 10).Radius(10).
		Border(0.5, border).Background(background).Selected(checked).Focusable()
	if multi {
		card.Role(ui.RoleCheckBox)
	} else {
		card.Role(ui.RoleRadio)
	}
	if disabled {
		card.Disabled(true)
	}
	card.Children(func() {
		badge := ui.Box(c).Size(questionOptionBadge, questionOptionBadge).Radius(6).Background(t.Surface).Center()
		badge.Children(func() {
			if checked {
				ui.Icon(c, questionOptionCheckIcon).Width(12).Height(12).TextColor(t.Accent)
				return
			}
			ui.Text(c, strconv.Itoa(index+1)).FontSize(11).TextColor(t.TextMuted)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).MinWidth(0).Children(func() {
				ui.Text(c, label).FontSize(13).FixedLineHeight(20).TextColor(t.Text).MinWidth(0).Shrink(1)
				if recommended {
					ui.Text(c, i18n.T(i18n.KeyChatQuestionRecommended)).FontSize(10).TextColor(t.Accent).
						Padding(2, 6, 2, 6).Radius(9).Background(t.Accent.Alpha(.12)).Shrink(0)
				}
			})
			if description != "" {
				ui.Text(c, description).FontSize(11).FixedLineHeight(17).TextColor(t.TextMuted).MinWidth(0)
			}
		})
	})
	if disabled || onToggle == nil {
		return card
	}
	if card.Clicked() {
		onToggle()
	}
	return card
}
