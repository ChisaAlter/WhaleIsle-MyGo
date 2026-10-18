package dshbot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Match the original schedule.js Host grammar, including 1–525600 minutes.
// This builds a schedule, never a local estimate of nextRunAt.
var routineCreateDurationPattern = regexp.MustCompile(`^(\d+)([mhd])$`)

func routineCreateDuration(amount, unit string) (string, error) {
	raw := strings.TrimSpace(amount) + unit
	if len(raw) > 200 || !routineCreateDurationPattern.MatchString(raw) {
		return "", fmt.Errorf("时长须为正整数和 m/h/d 单位 / Duration must be a positive integer with m/h/d")
	}
	n, err := strconv.ParseUint(strings.TrimSpace(amount), 10, 64)
	multiplier := map[string]uint64{"m": 1, "h": 60, "d": 1440}[unit]
	if err != nil || multiplier == 0 || n < 1 || n > 525600/multiplier {
		return "", fmt.Errorf("时长须为 1–525600 分钟（最多 365 天） / Duration must be 1–525600 minutes")
	}
	return strconv.FormatUint(n, 10) + unit, nil
}

func routineCreateSchedule(d RoutineCreateDraft) (string, error) {
	switch d.Frequency {
	case "", "daily":
		hour, hErr := strconv.Atoi(strings.TrimSpace(d.Hour))
		minute, mErr := strconv.Atoi(strings.TrimSpace(d.Minute))
		if hErr != nil || mErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return "", fmt.Errorf("请填写有效时间（00–23时，00–59分） / Enter a valid daily time")
		}
		return fmt.Sprintf("%d %d * * *", minute, hour), nil
	case "once":
		raw := strings.ToLower(strings.TrimSpace(d.OnceDelay))
		match := routineCreateDurationPattern.FindStringSubmatch(raw)
		if len(raw) > 200 || len(match) != 3 {
			return "", fmt.Errorf("一次性延迟格式须为 30m、1h 或 1d / Enter a one-shot delay such as 30m, 1h or 1d")
		}
		return routineCreateDuration(match[1], match[2])
	case "interval":
		duration, err := routineCreateDuration(d.IntervalAmount, d.IntervalUnit)
		if err != nil {
			return "", err
		}
		return "every " + duration, nil
	default:
		return routineCreateCalendarSchedule(d)
	}
}
