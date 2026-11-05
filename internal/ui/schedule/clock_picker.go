package schedule

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
)

type scheduleClockPicker struct {
	open, focus, reveal bool
	column, index       int
	refs                [3][60]ui.Handle
}

func schedulePickerTrigger(c *ui.Context, key, label, value string, open bool) ui.Element {
	trigger := ui.Button(c.Key(key), "").Label(label).Role(ui.RolePopUpButton).Expanded(open).Value(value).FillWidth().MinWidth(0).Height(48).Padding(0).Border(.5, c.Theme().Border).BorderWidth(0, 0, .5, 0).Radius(0).Background(ui.Transparent)
	trigger.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(14)
			ui.Box(c).Grow(1)
			kind := ""
			if strings.Contains(key, "date") {
				kind = "date"
			}
			if strings.Contains(key, "clock") {
				kind = "clock"
			}
			scheduleValueFace(c, value, open, kind)
		})
	})
	return trigger
}

func scheduleClockBase(clock string) string {
	parts := strings.SplitN(clock, ".", 2)
	fields := strings.Split(parts[0], ":")
	if len(fields) == 2 {
		fields = append(fields, "00")
	}
	if len(fields) != 3 {
		return "00:00:00"
	}
	for column, value := range fields {
		n, err := strconv.Atoi(value)
		limit := 60
		if column == 0 {
			limit = 24
		}
		if err != nil || len(value) != 2 || n < 0 || n >= limit {
			return "00:00:00"
		}
	}
	return strings.Join(fields, ":")
}

// Picks stay in the clock, with one scrolling, roving-focus entry per column.
// Untouched milliseconds remain in the caller's draft; only a pick truncates them.
func clockPicker(c *ui.Context, state *scheduleClockPicker, clock string, seconds bool, onPick func(string)) {
	base := scheduleClockBase(clock)
	face := base
	columns := 3
	if !seconds {
		columns, face = 2, base[:5]
	}
	label := scheduleText("时间", "Time")
	trigger := schedulePickerTrigger(c, "schedule-clock-trigger", label, face, state.open)
	values := []int{}
	for _, part := range strings.Split(base, ":") {
		value, _ := strconv.Atoi(part)
		values = append(values, value)
	}
	if trigger.Clicked() {
		state.open = !state.open
		state.column, state.index = 0, values[0]
		state.focus, state.reveal = state.open, state.open
	}
	ui.PopoverBase(c.Key("schedule-clock-popup"), trigger, &state.open, func(panel ui.Element) {
		panel = chrome.MenuSurface(panel.Width(float32(columns*52+(columns-1)*2+6)).MaxWidthPercent(100).Padding(3).Role(ui.RoleDialog).Label(label).Margin(4, 0, 0, 0), c.Theme().Background, ui.Transparent, 16)
		if panel.OverlayShortcut(0, ui.KeyEscape) {
			state.open = false
			trigger.Focus()
			return
		}
		panel.Children(func() {
			ui.Row(c).FillWidth().Gap(2).Children(func() {
				for column := 0; column < columns; column++ {
					column := column
					limit := 60
					if column == 0 {
						limit = 24
					}
					name := []string{scheduleText("时", "Hour"), scheduleText("分", "Minute"), scheduleText("秒", "Second")}[column]
					ui.Column(c).Width(52).MinWidth(0).Children(func() {
						ui.Scroll(c.Key(fmt.Sprintf("schedule-clock-column-%d", column))).Role(ui.RoleList).Label(name).MaxHeight(216).FillWidth().Children(func() {
							for index := 0; index < limit; index++ {
								index := index
								value := fmt.Sprintf("%02d", index)
								option := ui.Box(c.Key(value)).Bind(&state.refs[column][index]).Role(ui.RoleListItem).Label(name + " " + value).Checked(values[column] == index).FillWidth().Height(32).Center().Radius(10)
								cursor := state.column == column && state.index == index
								if cursor || state.column != column && values[column] == index {
									option.Focusable()
								}
								if cursor && state.focus {
									option.Focus().ScrollIntoView()
									state.focus = false
								}
								if state.reveal && values[column] == index {
									option.ScrollIntoView()
								}
								ink := c.Theme().Text
								if values[column] == index {
									option.Background(c.Theme().SurfaceHover).Border(1, c.Theme().Border)
								} else if option.Hovered() {
									option.Background(c.Theme().SurfaceHover)
								}
								option.Children(func() { ui.Text(c, value).TextAlign(ui.Center).FontSize(13).TextColor(ink) })
								if option.Focused() {
									state.column, state.index = column, index
								}
								nextColumn, nextIndex := column, index
								if option.Shortcut(0, ui.KeyUp) {
									nextIndex = max(0, index-1)
								} else if option.Shortcut(0, ui.KeyDown) {
									nextIndex = min(limit-1, index+1)
								} else if option.Shortcut(0, ui.KeyHome) {
									nextIndex = 0
								} else if option.Shortcut(0, ui.KeyEnd) {
									nextIndex = limit - 1
								} else if option.Shortcut(0, ui.KeyLeft) {
									nextColumn = max(0, column-1)
								} else if option.Shortcut(0, ui.KeyRight) {
									nextColumn = min(columns-1, column+1)
								}
								if nextColumn == 0 {
									nextIndex = min(nextIndex, 23)
								}
								if nextColumn != column || nextIndex != index {
									state.column, state.index = nextColumn, nextIndex
									state.focus = true
									state.refs[nextColumn][nextIndex].RequestFocus(c)
								}
								if option.Clicked() || option.Shortcut(0, ui.KeyEnter) || option.Shortcut(0, ui.KeySpace) {
									state.column, state.index = column, index
									option.Focus()
									onPick(clockPick(base, column, value))
								}
							}
						})
					})
				}
			})
			state.reveal = false
		})
	})
}
