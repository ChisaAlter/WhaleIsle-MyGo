package schedule

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/paths"
)

type timingPickerState struct {
	calendarOpen bool
	month        time.Time
	rawCron      bool
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
	return instant.Format("2006-01-02"), instant.Format("15:04:05")
}
func (d *taskDraft) pickWallClock(date, clock, zone string) error {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return fmt.Errorf("choose a valid IANA time zone")
	}
	wall := date + "T" + clock
	instant, err := time.ParseInLocation("2006-01-02T15:04:05", wall, loc)
	if err != nil || instant.Format("2006-01-02T15:04:05") != wall {
		return fmt.Errorf("date/time does not exist in %s", zone)
	}
	d.At = instant.Format(time.RFC3339)
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
func clockPicker(c *ui.Context, clock string, seconds bool, onPick func(string)) {
	labels := []string{"时 / Hour", "分 / Minute", "秒 / Second"}
	count := 3
	if !seconds {
		count = 2
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		for col := 0; col < count; col++ {
			n := 60
			if col == 0 {
				n = 24
			}
			options := make([]string, n)
			for i := range options {
				options[i] = fmt.Sprintf("%02d", i)
			}
			value := "00"
			if len(clock) >= col*3+2 {
				value = clock[col*3 : col*3+2]
			}
			ui.Column(c).Gap(3).Children(func() {
				ui.Text(c, labels[col])
				if ui.Select(c, &value, options).Changed() {
					onPick(clockPick(clock, col, value))
				}
			})
		}
	})
}
func (p *Panel) zonePicker(c *ui.Context, d *taskDraft) {
	if p.recentZones == nil {
		p.recentZones = loadRecentZones(systemScheduleZone())
	}
	selectZone := func(zone string) {
		if _, err := time.LoadLocation(zone); err != nil {
			p.message("请输入有效 IANA 时区 / Choose a valid IANA zone")
			return
		}
		if d.Kind == "at" {
			date, clock := d.wallClock()
			if err := d.pickWallClock(date, clock, zone); err != nil {
				p.message(err.Error())
				return
			}
		} else {
			d.Zone = zone
		}
		p.recentZones = rememberZone(loadRecentZones(systemScheduleZone()), zone)
	}
	ui.Text(c, "时区 / Time zone: "+d.Zone)
	ui.Column(c).Gap(4).Children(func() {
		for _, zone := range p.recentZones {
			if ui.Button(c, zone).Clicked() {
				selectZone(zone)
			}
		}
	})
	zone := d.Zone
	if ui.Combobox(c, &zone, scheduleZoneNames).Label("搜索时区 / Search time zones").FillWidth().Changed() {
		selectZone(zone)
	}

}
func (p *Panel) datePicker(c *ui.Context, d *taskDraft) {
	date, clock := d.wallClock()
	if ui.Button(c, "日期 / Date: "+date).Clicked() {
		d.picker.calendarOpen = !d.picker.calendarOpen
		selected, _ := time.Parse("2006-01-02", date)
		d.picker.month = time.Date(selected.Year(), selected.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if d.picker.calendarOpen {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
			if ui.Button(c, "‹").Clicked() {
				d.picker.month = d.picker.month.AddDate(0, -1, 0)
			}
			ui.Text(c, d.picker.month.Format("2006-01"))
			if ui.Button(c, "›").Clicked() {
				d.picker.month = d.picker.month.AddDate(0, 1, 0)
			}
			if ui.Button(c, "关闭 / Close").Clicked() {
				d.picker.calendarOpen = false
			}
		})
		ui.Row(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
			for _, label := range []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"} {
				ui.Text(c, label).Grow(1).MinWidth(0).TextAlign(ui.Center).FontSize(11)
			}
		})
		cells := calendarCells(d.picker.month)
		for row := 0; row < len(cells); row += 7 {
			ui.Row(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
				for _, day := range cells[row : row+7] {
					if day == 0 {
						ui.Text(c, "").Grow(1).MinWidth(0)
						continue
					}
					picked := fmt.Sprintf("%04d-%02d-%02d", d.picker.month.Year(), d.picker.month.Month(), day)
					label := strconv.Itoa(day)
					if picked == date {
						label = "[" + label + "]"
					}
					if ui.Button(c, label).Grow(1).MinWidth(0).Padding(0).Height(28).Clicked() {
						if err := d.pickWallClock(picked, clock, d.Zone); err != nil {
							p.message(err.Error())
						} else {
							d.picker.calendarOpen = false
						}
					}
				}
			})
		}
	}
	clockPicker(c, clock, true, func(value string) {
		if err := d.pickWallClock(date, value, d.Zone); err != nil {
			p.message(err.Error())
		}
	})
}
func dayPills(c *ui.Context, days []int, max int, onPick func([]int)) {
	for start := 1; start <= max; start += 7 {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(4).Children(func() {
			for day := start; day < start+7 && day <= max; day++ {
				label := strconv.Itoa(day)
				if max == 7 {
					label = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}[day-1]
				}
				for _, selected := range days {
					if selected == day {
						label = "[" + label + "]"
						break
					}
				}
				if ui.Button(c, label).Clicked() {
					onPick(toggleNumber(days, day))
				}
			}
		})
	}
}
func (p *Panel) timingEditor(c *ui.Context, d *taskDraft, editable bool) {
	if !editable {
		d.picker.calendarOpen = false
		ui.Text(c, "当前时间规则只读 / Timing is read-only")
		return
	}
	if ui.Select(c, &d.Kind, []string{"at", "every", "daily", "weekly", "cron"}).Changed() {
		d.picker = timingPickerState{}
		_, clock := d.wallClock()
		if d.Clock == "" {
			d.Clock = clock
		}
		if d.Seconds == "0" || d.Seconds == "" {
			d.Seconds = "60"
		}
		if d.Weekdays == "" {
			d.Weekdays = "1"
		}
		if d.Expression == "" {
			h, _ := strconv.Atoi(clock[:2])
			m, _ := strconv.Atoi(clock[3:5])
			d.Expression = cronShape{kind: "daily", hour: h, minute: m}.expression()
		}
	}
	switch d.Kind {
	case "at":
		p.datePicker(c, d)
		p.zonePicker(c, d)
	case "every":
		ui.Text(c, "间隔秒数 / Interval seconds")
		ui.TextInput(c, &d.Seconds).FillWidth()
	case "daily", "weekly":
		clockPicker(c, d.Clock, true, func(value string) { d.Clock = value })
		p.zonePicker(c, d)
		if d.Kind == "weekly" {
			days := []int{}
			for _, part := range strings.Split(d.Weekdays, ",") {
				if n, err := strconv.Atoi(part); err == nil {
					days = append(days, n)
				}
			}
			dayPills(c, days, 7, func(values []int) {
				parts := []string{}
				for _, n := range values {
					parts = append(parts, strconv.Itoa(n))
				}
				d.Weekdays = strings.Join(parts, ",")
			})
		}
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
	if ui.Select(c, &choice, []string{"monthly", "weekly", "daily", "hourly", "minutely", "raw"}).Changed() {
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
		ui.TextInput(c, &d.Expression).FillWidth().Placeholder("minute hour day month weekday")
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
			ui.Text(c, "每隔 / Every")
			if ui.Select(c, &step, options).Changed() {
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
			ui.Text(c, "分钟 / Minute")
			if ui.Select(c, &minute, options).Changed() {
				shape.minute, _ = strconv.Atoi(minute)
				update()
			}
		}
		if choice == "daily" || choice == "weekly" || choice == "monthly" {
			clockPicker(c, fmt.Sprintf("%02d:%02d:00", shape.hour, shape.minute), false, func(value string) {
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
	if parsed, err := parseTaskCron(d.Expression); err != nil {
		ui.Text(c, err.Error())
	} else {
		ui.Text(c, "预览 / Preview: "+parsed.preview()+" · "+d.Zone)
	}
}
