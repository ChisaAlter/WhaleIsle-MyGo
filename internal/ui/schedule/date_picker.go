package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
	"whaleisle/internal/ui/widgets"
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
	t := design.For(c)
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
	panel := coss.Popover(c.Key("schedule-date-popup"), trigger, &state.calendarOpen, coss.PopoverOptions{
		Side:  coss.SideBottom,
		Align: coss.AlignStart,
		Width: 248,
	}, func() {
		ui.Column(c).FillWidth().Gap(design.Sp(1)).Children(func() {
			ui.Row(c).FillWidth().Gap(design.Sp(1)).AlignItems(ui.Center).Children(func() {
				if coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconXS,
					Icon:    coss.IconChevronLeft,
					Label:   scheduleText("上个月", "Previous month"),
				}).Clicked() {
					scheduleShiftMonth(state, -1)
				}
				pagekit.Strong(c, scheduleMonthTitle(state.month)).Grow(1).TextAlign(ui.Center)
				if coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconXS,
					Icon:    coss.IconChevronRight,
					Label:   scheduleText("下个月", "Next month"),
				}).Clicked() {
					scheduleShiftMonth(state, 1)
				}
			})
			grid := ui.Column(c.Key("schedule-date-grid")).Role(ui.RoleTable).Label(scheduleMonthTitle(state.month)).FillWidth().Gap(design.Sp(0.5))
			month := state.month
			grid.Children(func() {
				ui.Row(c).FillWidth().Gap(design.Sp(0.5)).Children(func() {
					for day := 1; day <= 7; day++ {
						pagekit.Caption(c, catalogWeekday(day, i18n.Locale())).Role(ui.RoleColumnHeader).Grow(1).MinWidth(0).Height(design.Sp(5.5)).TextAlign(ui.Center)
					}
				})
				cells := calendarCells(month)
				last := month.AddDate(0, 1, -1).Day()
				for row := 0; row < len(cells); row += 7 {
					ui.Row(c).Role(ui.RoleRow).FillWidth().Gap(design.Sp(0.5)).Children(func() {
						for _, day := range cells[row : row+7] {
							if day == 0 {
								ui.Box(c).Grow(1).MinWidth(0).Height(design.Sp(7)).Role(ui.RoleNone)
								continue
							}
							day := day
							iso := fmt.Sprintf("%04d-%02d-%02d", month.Year(), month.Month(), day)
							cell := ui.Box(c.Key("schedule-calendar-" + iso)).Bind(&state.calendarCells[day]).
								Grow(1).MinWidth(0).Height(design.Sp(7)).Center().Radius(design.RadiusMD).
								Role(ui.RoleCell).Label(iso).Checked(iso == date).Cursor(ui.CursorPointer)
							if day == state.calendarDay {
								cell.Focusable()
								if state.calendarFocus {
									cell.Focus()
									state.calendarFocus = false
								}
							}
							ink := t.Foreground
							if iso == date {
								cell.Background(t.Accent).Border(design.BorderWidth, t.Border)
							} else if cell.Hovered() {
								cell.Background(t.Accent)
							}
							if iso == time.Now().Format("2006-01-02") && iso != date {
								cell.Border(design.BorderWidth, t.Border)
							}
							cell.Children(func() {
								ui.Text(c, strconv.Itoa(day)).TextAlign(ui.Center).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).TextColor(ink)
							})
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
	if panel != (ui.Element{}) {
		panel.MaxWidthPercent(100).Role(ui.RoleDialog).Label(label)
		panel.Opacity(widgets.Reveal(c, panel, design.TransitionFor("popover/popover-popup").Duration))
		if panel.OverlayShortcut(0, ui.KeyEscape) {
			state.calendarOpen = false
			trigger.Focus()
		}
	}
}
