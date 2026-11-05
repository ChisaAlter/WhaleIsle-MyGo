package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

func scheduleMonthTitle(month time.Time) string {
	if i18n.Locale() == i18n.LocaleZH {
		return fmt.Sprintf("%d年%d月", month.Year(), month.Month())
	}
	return month.Format("January 2006")
}

func scheduleShiftMonth(state *timingPickerState, step int) {
	state.month = state.month.AddDate(0, step, 0)
	last := state.month.AddDate(0, 1, -1).Day()
	state.calendarDay = min(max(1, state.calendarDay), last)
	state.calendarFocus = true
}

// The panel stages a day and stays open, matching the installed DatePicker.
// Navigation changes only the focused day, never the draft until a day is picked.
func (p *Panel) datePicker(c *ui.Context, d *taskDraft) {
	date, clock := d.wallClock()
	state := &d.picker
	label := scheduleText("日期", "Date")
	trigger := schedulePickerTrigger(c, "schedule-date-trigger", label, strings.ReplaceAll(date, "-", "/"), state.calendarOpen)
	if trigger.Clicked() {
		state.calendarOpen = !state.calendarOpen
		if state.calendarOpen {
			selected, err := time.Parse("2006-01-02", date)
			if err != nil {
				selected = time.Now()
			}
			state.month = time.Date(selected.Year(), selected.Month(), 1, 0, 0, 0, 0, time.UTC)
			state.calendarDay, state.calendarFocus = selected.Day(), true
			state.clock.open, state.zone.open = false, false
		}
	}
	ui.PopoverBase(c.Key("schedule-date-popup"), trigger, &state.calendarOpen, func(panel ui.Element) {
		panel = chrome.MenuSurface(panel.Width(232).MaxWidthPercent(100).Padding(8).Gap(4).Role(ui.RoleDialog).Label(label).Margin(4, 0, 0, 0), c.Theme().Background, ui.Transparent, 16)
		if panel.OverlayShortcut(0, ui.KeyEscape) {
			state.calendarOpen = false
			trigger.Focus()
			return
		}
		panel.Children(func() {
			ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "‹").Label(scheduleText("上个月", "Previous month")).Size(26, 26).Padding(0).BorderWidth(0).Radius(8).Clicked() {
					scheduleShiftMonth(state, -1)
				}
				ui.Text(c, scheduleMonthTitle(state.month)).Grow(1).TextAlign(ui.Center).FontSize(13).FontWeight(500)
				if ui.Button(c, "›").Label(scheduleText("下个月", "Next month")).Size(28, 28).Padding(0).BorderWidth(0).Clicked() {
					scheduleShiftMonth(state, 1)
				}
			})
			grid := ui.Column(c.Key("schedule-date-grid")).Role(ui.RoleTable).Label(scheduleMonthTitle(state.month)).FillWidth().Gap(2)
			month := state.month
			grid.Children(func() {
				ui.Row(c).FillWidth().Gap(2).Children(func() {
					for day := 1; day <= 7; day++ {
						ui.Text(c, catalogWeekday(day, i18n.Locale())).Role(ui.RoleColumnHeader).Grow(1).MinWidth(0).Height(22).TextAlign(ui.Center).FontSize(11).TextColor(c.Theme().TextMuted)
					}
				})
				cells := calendarCells(month)
				last := month.AddDate(0, 1, -1).Day()
				for row := 0; row < len(cells); row += 7 {
					ui.Row(c).Role(ui.RoleRow).FillWidth().Gap(2).Children(func() {
						for _, day := range cells[row : row+7] {
							if day == 0 {
								ui.Box(c).Grow(1).MinWidth(0).Height(28).Role(ui.RoleNone)
								continue
							}
							day := day
							iso := fmt.Sprintf("%04d-%02d-%02d", month.Year(), month.Month(), day)
							cell := ui.Box(c.Key("schedule-calendar-" + iso)).Bind(&state.calendarCells[day]).Grow(1).MinWidth(0).Height(28).Center().Radius(8).Role(ui.RoleCell).Label(iso).Checked(iso == date)
							if day == state.calendarDay {
								cell.Focusable()
								if state.calendarFocus {
									cell.Focus()
									state.calendarFocus = false
								}
							}
							ink := c.Theme().Text
							if iso == date {
								cell.Background(c.Theme().SurfaceHover).Border(1, c.Theme().Border)
							} else if cell.Hovered() {
								cell.Background(c.Theme().SurfaceHover)
							}
							if iso == time.Now().Format("2006-01-02") && iso != date {
								cell.Border(.5, c.Theme().Border)
							}
							cell.Children(func() { ui.Text(c, strconv.Itoa(day)).TextAlign(ui.Center).FontSize(13).TextColor(ink) })
							next := day
							if cell.Shortcut(0, ui.KeyLeft) {
								next--
							} else if cell.Shortcut(0, ui.KeyRight) {
								next++
							} else if cell.Shortcut(0, ui.KeyUp) {
								next -= 7
							} else if cell.Shortcut(0, ui.KeyDown) {
								next += 7
							} else if cell.Shortcut(0, ui.KeyHome) {
								next = 1
							} else if cell.Shortcut(0, ui.KeyEnd) {
								next = last
							}
							if next != day {
								state.calendarDay = min(max(1, next), last)
								state.calendarCells[state.calendarDay].RequestFocus(c)
							}
							if cell.Shortcut(0, ui.KeyPageUp) {
								scheduleShiftMonth(state, -1)
							} else if cell.Shortcut(0, ui.KeyPageDown) {
								scheduleShiftMonth(state, 1)
							}
							if cell.Clicked() || cell.Shortcut(0, ui.KeyEnter) || cell.Shortcut(0, ui.KeySpace) {
								state.calendarDay = day
								cell.Focus()
								if err := d.pickWallClock(iso, clock, d.Zone); err != nil {
									p.message(scheduleText("该日期和时间在所选时区中不存在，请调整后重试。", "This date and time do not exist in the selected time zone. Adjust them and try again."))
								}
							}
						}
					})
				}
			})
		})
	})
}
