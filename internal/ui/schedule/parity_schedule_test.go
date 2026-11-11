package schedule

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestScheduledTasksCatalogAndFormattingParity(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	i18n.SetLocale(i18n.LocaleZH)
	zhChecks := map[string]string{
		"trigger.label":         "提醒",
		"list.nextRun":          "下次提醒：{time}",
		"frequency.daily":       "每天 {time}",
		"frequency.weekly":      "每周{weekday} {time}",
		"frequency.cronRule":    "{rule}（{timeZone}）",
		"frequency.cronLocal":   "{rule}",
		"cron.day.every":        "每天{months}",
		"cron.day.monthdays":    "每月{days}日{months}",
		"cron.day.both":         "每月{days}日或{weekdays}{months}",
		"cron.hours.every":      "每小时",
		"cron.hours.step":       "每 {step} 小时",
		"cron.time.everyMinute": "{hours}每分钟",
		"cron.time.stepMinutes": "{hours}每 {step} 分钟",
		"cron.time.atMinutes":   "{hours}第 {minutes} 分",
		"relative.now":          "即将",
		"relative.past":         "{value}{unit}前",
		"relative.future":       "{value}{unit}后",
	}
	for k, want := range zhChecks {
		if got := ScheduleLocaleText(k); got != want {
			t.Fatalf("zh ScheduleLocaleText(%q) = %q, want %q", k, got, want)
		}
	}

	i18n.SetLocale(i18n.LocaleEN)
	enChecks := map[string]string{
		"trigger.label":         "Reminders",
		"list.nextRun":          "Next reminder: {time}",
		"frequency.daily":       "Daily at {time}",
		"frequency.weekly":      "Weekly on {weekday} at {time}",
		"frequency.cronRule":    "{rule} ({timeZone})",
		"frequency.cronLocal":   "{rule}",
		"cron.day.every":        "Every day{months}",
		"cron.day.monthdays":    "Day {days} of the month{months}",
		"cron.day.both":         "Day {days} or {weekdays}{months}",
		"cron.hours.every":      "every hour",
		"cron.hours.step":       "every {step} hours",
		"cron.time.everyMinute": "every minute ({hours})",
		"cron.time.stepMinutes": "every {step} minutes ({hours})",
		"cron.time.atMinutes":   "at minute {minutes} ({hours})",
		"relative.now":          "due now",
		"relative.past":         "{value} {unit} ago",
		"relative.future":       "in {value} {unit}",
	}
	for k, want := range enChecks {
		if got := ScheduleLocaleText(k); got != want {
			t.Fatalf("en ScheduleLocaleText(%q) = %q, want %q", k, got, want)
		}
	}
}

func TestScheduleCatalogViewHeadlessAndCursors(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	snap := schedstore.ViewState{
		ClientReady: true,
		Rows: []schedstore.Entry{
			{
				ID:     "rem-1",
				Title:  "Daily Standup",
				Status: "active",
				Raw:    json.RawMessage(`{"kind":"daily","time":"09:00"}`),
			},
		},
	}

	for _, loc := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(loc)
		var panel Panel
		tt := ui.NewTester(func(c *ui.Context) {
			colors := theme.Resolve("system").Active(false)
			panel.catalogView(c, Model{
				ThemeID:   "system",
				OnNewTask: func() {},
			}, snap, colors, 760)
		}, 760, 480)

		if !tt.HasText("Daily Standup") {
			t.Fatalf("locale %s missing task title in %v", loc, tt.Texts())
		}

		for _, it := range tt.Interactives() {
			if !slices.Contains(it.Kinds, "click") || slices.Contains(it.Kinds, "text") {
				continue
			}
			if it.Cursor != ui.CursorPointer {
				t.Fatalf("locale %s interactive %q cursor = %v, want CursorPointer", loc, it.Label, it.Cursor)
			}
		}
	}
}
