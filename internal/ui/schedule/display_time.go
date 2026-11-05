package schedule

import (
	"fmt"
	"math"
	"time"

	"whaleisle/internal/ui/i18n"
)

// Mirrors schedule-format.nextRunParts: this is the Host's committed instant,
// displayed in the device zone, with the same instant used for the countdown.
func scheduleAbsolute(at time.Time) string {
	if i18n.Locale() == i18n.LocaleZH {
		return fmt.Sprintf("%d月%d日 %s", at.Month(), at.Day(), at.Format("15:04"))
	}
	return at.Format("Jan 2, 2006, 15:04")
}

func scheduleRelative(at, now time.Time) string {
	difference := at.Sub(now).Seconds()
	if difference == 0 {
		return scheduleText("现在", "due now")
	}
	absolute := math.Abs(difference)
	for _, unit := range []struct {
		seconds float64
		zh, en  string
	}{{86400, "天", "day"}, {3600, "小时", "hour"}, {60, "分钟", "minute"}, {1, "秒", "second"}} {
		if absolute < unit.seconds && unit.seconds != 1 {
			continue
		}
		value := math.Floor(absolute / unit.seconds)
		if difference > 0 {
			value = math.Ceil(absolute / unit.seconds)
		}
		value = math.Max(1, value)
		name := unit.en
		if value != 1 {
			name += "s"
		}
		if difference > 0 {
			return fmt.Sprintf(scheduleText("%.0f%s后", "in %.0f %s"), value, scheduleText(unit.zh, name))
		}
		return fmt.Sprintf(scheduleText("已过期 %.0f%s", "%.0f %s overdue"), value, scheduleText(unit.zh, name))
	}
	return ""
}
