package schedule

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

type timingPickerState struct {
	calendarOpen  bool
	month         time.Time
	calendarDay   int
	calendarFocus bool
	calendarCells [32]ui.Handle
	rawCron       bool
	clock         scheduleClockPicker
	zone          scheduleZonePicker
}

func calendarCells(month time.Time) []int {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	offset := (int(first.Weekday()) + 6) % 7
	days := first.AddDate(0, 1, -1).Day()
	cells := make([]int, ((offset+days+6)/7)*7)
	for day := 1; day <= days; day++ {
		cells[offset+day-1] = day
	}
	return cells
}
func clockPick(clock string, column int, value string) string {
	base := "00:00:00"
	if len(clock) >= 8 {
		base = clock[:8]
	}
	parts := strings.Split(base, ":")
	parts[column] = value
	return strings.Join(parts, ":")
}
func (d *taskDraft) wallClock() (string, string) {
	loc, err := time.LoadLocation(d.Zone)
	if err != nil {
		loc = time.UTC
	}
	instant, err := time.Parse(time.RFC3339Nano, d.At)
	if err != nil {
		instant = time.Now()
	}
	instant = instant.In(loc)
	return instant.Format("2006-01-02"), instant.Format("15:04:05.999")
}
func (d *taskDraft) pickWallClock(date, clock, zone string) error {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return fmt.Errorf("choose a valid IANA time zone")
	}
	wall := date + "T" + clock
	instant, err := time.ParseInLocation("2006-01-02T15:04:05", wall, loc)
	// Parse accepts fractions even with a whole-second layout. Compare the wall
	// fields, not formatted fraction spelling, to reject DST gaps without
	// discarding an untouched stored millisecond value.
	if err != nil || instant.Format("2006-01-02T15:04:05") != strings.SplitN(wall, ".", 2)[0] {
		return fmt.Errorf("date/time does not exist in %s", zone)
	}
	d.At = instant.Format(time.RFC3339Nano)
	d.Zone = zone
	return nil
}

// Prefer the OS's IANA identity; never infer a zone from language or UTC offset.
// Windows uses the stable system key and the CLDR global IANA mapping.
// Unrecognized/custom systems retain UTC fallback and the explicit zone selector.
func systemScheduleZone() string {
	candidates := []string{strings.TrimPrefix(os.Getenv("TZ"), ":"), platformScheduleZone(), time.Local.String()}
	if link, err := os.Readlink("/etc/localtime"); err == nil {
		if at := strings.LastIndex(link, "zoneinfo/"); at >= 0 {
			candidates = append(candidates, link[at+len("zoneinfo/"):])
		}
	}
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		candidates = append(candidates, strings.TrimSpace(string(data)))
	}
	for _, zone := range candidates {
		if zone != "UTC" && !strings.Contains(zone, "/") {
			continue
		}
		if _, err := time.LoadLocation(zone); err == nil {
			return zone
		}
	}
	return "UTC"
}

func recentZonePath() string {
	return filepath.Join(paths.UserDataDir(), "schedule-recent-time-zones.json")
}
func loadRecentZones(system string) []string {
	var values []string
	data, err := os.ReadFile(recentZonePath())
	if err == nil {
		_ = json.Unmarshal(data, &values)
	}
	result := []string{}
	for _, zone := range values {
		if _, err := time.LoadLocation(zone); err != nil || zone == "" {
			continue
		}
		if !containsZone(result, zone) {
			result = append(result, zone)
		}
		if len(result) == 5 {
			break
		}
	}
	if len(result) == 0 {
		result = []string{system}
		if system != "UTC" {
			result = append(result, "UTC")
		}
	}
	return result
}
func containsZone(zones []string, zone string) bool {
	for _, z := range zones {
		if z == zone {
			return true
		}
	}
	return false
}
func rememberZone(recent []string, zone string) []string {
	next := []string{zone}
	for _, z := range recent {
		if z != zone && !containsZone(next, z) {
			next = append(next, z)
		}
		if len(next) == 5 {
			break
		}
	}
	data, _ := json.Marshal(next)
	// As with browser-local recents, storage failures never block a schedule edit.
	if os.MkdirAll(filepath.Dir(recentZonePath()), 0700) == nil {
		_ = os.WriteFile(recentZonePath(), data, 0600)
	}
	return next
}
func dayPills(c *ui.Context, days []int, maximum int, onPick func([]int)) {
	for start := 1; start <= maximum; start += 7 {
		ui.Row(c).FillWidth().MinWidth(0).Justify(ui.End).Gap(4).Padding(6, 0).Children(func() {
			for day := start; day < start+7 && day <= maximum; day++ {
				day := day
				label := strconv.Itoa(day)
				if maximum == 7 {
					label = catalogWeekday(day, i18n.Locale())
				}
				selected := slices.Contains(days, day)
				button := ui.Button(c, label).Label(label).Selected(selected).MinWidth(28).Height(28).Padding(0, 6).Radius(14).BorderWidth(0).Background(ui.Transparent).FontSize(13)
				if selected {
					button.Background(c.Theme().SurfaceHover).Border(1, c.Theme().Border)
				}
				if button.Clicked() {
					onPick(toggleNumber(days, day))
				}
			}
		})
	}
}
func (p *Panel) timingEditor(c *ui.Context, d *taskDraft, editable bool) {
	if !editable {
		d.picker.calendarOpen = false
		d.picker.clock.open = false
		d.picker.zone.open = false
	}
	ui.Column(c.Key("schedule-timing-editor")).FillWidth().MinWidth(0).Gap(0).Disabled(!editable).Children(func() {
		p.timingEditorBody(c, d)
	})
}

func (p *Panel) timingEditorBody(c *ui.Context, d *taskDraft) {
	choice := d.Kind
	if choice == "every" {
		switch d.intervalUnit {
		case 3600:
			choice = "every-hour"
		case 60:
			choice = "every-minute"
		default:
			choice = "every-second"
		}
	} else if choice == "weekly" && d.weekdayShortcut {
		choice = "weekdays"
	}
	if scheduleChoice(c, "schedule-rule-repeat", scheduleText("重复", "Repeat"), &choice, []scheduleChoiceOption{
		{"at", scheduleText("仅一次", "Once")},
		{"every-second", scheduleText("每隔若干秒", "Every N seconds")},
		{"every-minute", scheduleText("每隔若干分钟", "Every N minutes")},
		{"every-hour", scheduleText("每隔若干小时", "Every N hours")},
		{"daily", scheduleText("每天", "Every day")},
		{"weekdays", scheduleText("周一至周五", "Monday to Friday")},
		{"weekly", scheduleText("每周", "Weekly")},
		{"cron", scheduleText("自定义", "Custom")},
	}) {
		switch choice {
		case "every-second", "every-minute", "every-hour":
			d.chooseKind("every")
			d.intervalUnit = 1
			if choice == "every-minute" {
				d.intervalUnit = 60
			}
			if choice == "every-hour" {
				d.intervalUnit = 3600
			}
		case "weekdays":
			d.chooseKind("weekly")
			d.Weekdays = "1,2,3,4,5"
			d.weekdayShortcut = true
		default:
			d.chooseKind(choice)
			d.weekdayShortcut = false
		}
	}
	switch d.Kind {
	case "at":
		p.datePicker(c, d)
		date, clock := d.wallClock()
		clockPicker(c, &d.picker.clock, clock, true, func(value string) {
			if err := d.pickWallClock(date, value, d.Zone); err != nil {
				p.message(scheduleText("该日期和时间在所选时区中不存在，请调整后重试。", "This date and time do not exist in the selected time zone. Adjust them and try again."))
			}
		})
		p.zonePicker(c, d)
	case "every":
		factor := max(1, d.intervalUnit)
		seconds, err := strconv.ParseFloat(d.Seconds, 64)
		quantity := d.Seconds
		if err == nil {
			quantity = strconv.FormatFloat(seconds/float64(factor), 'f', -1, 64)
		}
		scheduleRuleRow(c, scheduleText("间隔", "Interval"), true, func() {
			stepper := ui.Row(c).MinWidth(72).Height(32).Radius(16).Gap(2).AlignItems(ui.Center)
			if stepper.Hovered() || stepper.FocusWithin() {
				stepper.Background(c.Theme().SurfaceHover)
			}
			stepper.Children(func() {
				if ui.TextInputBase(c, &quantity).Label("Task repeat interval").Width(max(float32(72), float32(len(quantity))*8+52)).Height(32).Padding(0, 26).TextAlign(ui.Center).FontSize(14).Changed() {
					if value, err := strconv.ParseFloat(quantity, 64); err == nil && !math.IsNaN(value) && !math.IsInf(value, 0) {
						d.Seconds = strconv.FormatFloat(math.Round(value*float64(factor)), 'f', 0, 64)
					} else {
						d.Seconds = quantity
					}
				}
				if stepper.Hovered() || stepper.FocusWithin() {
					ui.Column(c).Absolute().Right(8).Gap(2).Children(func() {
						for _, delta := range []int{1, -1} {
							delta := delta
							label := scheduleText("增加间隔", "Increase interval")
							if delta < 0 {
								label = scheduleText("减少间隔", "Decrease interval")
							}
							button := ui.Button(c, "").Label(label).Size(17, 12).Padding(0).Radius(3).BorderWidth(0).Background(c.Theme().Surface)
							button.Children(func() {
								icon := ui.Icon(c, scheduleChevron).Size(9, 9)
								if delta > 0 {
									icon.Rotate(180)
								}
							})
							if button.Clicked() {
								value, _ := strconv.Atoi(d.Seconds)
								d.Seconds = strconv.Itoa(max(factor, 60, value+delta*factor))
							}
						}
					})
				}
			})
			unit := scheduleText("秒", "seconds")
			if factor == 60 {
				unit = scheduleText("分钟", "minutes")
			}
			if factor == 3600 {
				unit = scheduleText("小时", "hours")
			}
			ui.Text(c, unit).FontSize(14)
		})

	case "daily", "weekly":
		if d.Kind == "weekly" {
			days := []int{}
			for _, part := range strings.Split(d.Weekdays, ",") {
				if n, err := strconv.Atoi(part); err == nil {
					days = append(days, n)
				}
			}
			scheduleRuleRow(c, scheduleText("星期", "Weekdays"), false, func() {
				ui.Column(c).MinWidth(0).MaxWidthPercent(80).Children(func() {
					dayPills(c, days, 7, func(values []int) {
						d.weekdayShortcut = false
						parts := []string{}
						for _, n := range values {
							parts = append(parts, strconv.Itoa(n))
						}
						d.Weekdays = strings.Join(parts, ",")
					})
				})
			})
		}
		clockPicker(c, &d.picker.clock, d.Clock, true, func(value string) { d.Clock = value })
		p.zonePicker(c, d)

	case "cron":
		p.cronEditor(c, d)
		p.zonePicker(c, d)
	}
}
func (p *Panel) cronEditor(c *ui.Context, d *taskDraft) {
	parsed, err := parseTaskCron(d.Expression)
	shape, recognized := parsed.shape()
	choice := "raw"
	if err == nil && recognized && !d.picker.rawCron {
		choice = shape.kind
	}
	if scheduleChoice(c, "schedule-cron-frequency", scheduleText("频率", "Frequency"), &choice, []scheduleChoiceOption{
		{"monthly", scheduleText("每月", "Monthly")},
		{"weekly", scheduleText("每周", "Weekly")},
		{"daily", scheduleText("每天", "Every day")},
		{"hourly", scheduleText("每隔若干小时", "Every N hours")},
		{"minutely", scheduleText("每隔若干分钟", "Every N minutes")},
		{"raw", scheduleText("Cron 表达式", "Cron expression")},
	}) {
		d.picker.rawCron = choice == "raw"
		if choice != "raw" {
			date, clock := d.wallClock()
			instant, _ := time.Parse("2006-01-02", date)
			hour, _ := strconv.Atoi(clock[:2])
			minute, _ := strconv.Atoi(clock[3:5])
			if recognized {
				minute = shape.minute
				if shape.kind == "daily" || shape.kind == "weekly" || shape.kind == "monthly" {
					hour = shape.hour
				}
			}
			shape = cronShape{kind: choice, step: 1, hour: hour, minute: minute}
			if choice == "minutely" {
				shape.step = 5
			}
			if choice == "weekly" {
				day := int(instant.Weekday())
				if day == 0 {
					day = 7
				}
				shape.days = []int{day}
			}
			if choice == "monthly" {
				shape.days = []int{instant.Day()}
			}
			d.Expression = shape.expression()
		}
	}
	if choice == "raw" {
		scheduleRuleRow(c, scheduleText("表达式", "Expression"), false, func() {
			ui.TextInputBase(c, &d.Expression).Label("Cron expression").WidthPercent(70).MinWidth(0).Height(32).Padding(0, 8).Placeholder("minute hour day month weekday").FontSize(14)
		})
	} else {
		update := func() { d.Expression = shape.expression() }
		if choice == "minutely" || choice == "hourly" {
			max := 59
			if choice == "hourly" {
				max = 23
			}
			options := make([]string, max)
			for i := range options {
				options[i] = strconv.Itoa(i + 1)
			}
			step := strconv.Itoa(shape.step)
			choices := make([]scheduleChoiceOption, len(options))
			for i, v := range options {
				choices[i] = scheduleChoiceOption{v, v}
			}
			if scheduleChoice(c, "cron-step", i18n.T(i18n.KeyMiscScheduleEvery), &step, choices) {
				shape.step, _ = strconv.Atoi(step)
				update()
			}
		}
		if choice == "hourly" {
			minute := fmt.Sprintf("%02d", shape.minute)
			options := make([]string, 60)
			for i := range options {
				options[i] = fmt.Sprintf("%02d", i)
			}
			choices := make([]scheduleChoiceOption, len(options))
			for i, v := range options {
				choices[i] = scheduleChoiceOption{v, v}
			}
			if scheduleChoice(c, "cron-minute", i18n.T(i18n.KeyMiscScheduleMinute2), &minute, choices) {
				shape.minute, _ = strconv.Atoi(minute)
				update()
			}
		}
		if choice == "daily" || choice == "weekly" || choice == "monthly" {
			clockPicker(c, &d.picker.clock, fmt.Sprintf("%02d:%02d:00", shape.hour, shape.minute), false, func(value string) {
				shape.hour, _ = strconv.Atoi(value[:2])
				shape.minute, _ = strconv.Atoi(value[3:5])
				update()
			})
		}
		if choice == "weekly" || choice == "monthly" {
			max := 7
			if choice == "monthly" {
				max = 31
			}
			dayPills(c, shape.days, max, func(values []int) { shape.days = values; update() })
		}
		ui.Text(c, "Cron: "+d.Expression)
	}
	muted := c.Theme().TextMuted
	if parsed, err := parseTaskCron(d.Expression); err != nil {
		// Keep the parser's diagnostics internal; the editor names the accepted
		// shape and a usable example beside the field that produced it.
		ui.Text(c, scheduleText("请输入 5 字段的 cron 表达式，例如 0 9 * * 1-5", "Enter a five-field cron expression, for example 0 9 * * 1-5.")).FontSize(12).TextColor(c.Theme().Danger)
		return
	} else {
		ui.Text(c, i18n.T(i18n.KeyMiscSchedulePreview)+catalogCronPreview(parsed, i18n.Locale())+" · "+formatCatalogZone(d.Zone, time.Now(), i18n.Locale())).FontSize(12).TextColor(muted)
		loc := time.UTC
		if loaded, err := time.LoadLocation(d.Zone); err == nil {
			loc = loaded
		}
		if at, ok := parsed.nextAfter(time.Now(), loc); ok {
			ui.Text(c, scheduleText("下次计划时间：", "Next run: ")+at.Format("2006-01-02 15:04 MST")).FontSize(12).TextColor(muted)
		} else {
			ui.Text(c, scheduleText("这个表达式在 45 天内不会再运行。", "This expression does not run again within 45 days.")).FontSize(12).TextColor(muted)
		}
	}
}

// minuteMarks is the minute list of the wall-clock picker: quarter hours, plus
// whatever minute the expression already carries so editing never silently moves
// an existing task.
func minuteMarks(current string) []string {
	marks := []string{"00", "15", "30", "45"}
	if slices.Contains(marks, current) {
		return marks
	}
	all := append(append([]string{}, marks...), current)
	slices.Sort(all)
	return all
}
