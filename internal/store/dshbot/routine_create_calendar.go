package dshbot

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

func routineCreateFrequencySupported(frequency string) bool {
	switch frequency {
	case "once", "hourly", "daily", "weekdays", "weekly", "monthly", "interval", "advanced":
		return true
	}
	return false
}

// Mirror the original structured fields. The Host's cron-parser remains the
// authority on actual cron validity and month-end/DST next execution times.
func routineCreateCalendarSchedule(d RoutineCreateDraft) (string, error) {
	switch d.Frequency {
	case "hourly":
		return "every 1h", nil
	case "advanced":
		raw := strings.TrimSpace(d.Cron)
		fields := strings.Fields(raw)
		if len(fields) != 5 || len(utf16.Encode([]rune(raw))) > 200 || strings.ContainsRune(raw, '\x00') {
			return "", fmt.Errorf("Cron 须为五字段，最多 200 字符 / Cron needs five fields, at most 200 characters")
		}
		for _, field := range fields {
			if len(utf16.Encode([]rune(field))) > 40 {
				return "", fmt.Errorf("Cron 每字段最多 40 字符 / Each cron field may contain at most 40 characters")
			}
		}
		// Preserve the exact trimmed expression rather than guessing parser semantics.
		return raw, nil
	case "weekdays", "weekly", "monthly":
		hour, hErr := strconv.Atoi(strings.TrimSpace(d.Hour))
		minute, mErr := strconv.Atoi(strings.TrimSpace(d.Minute))
		if hErr != nil || mErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return "", fmt.Errorf("请填写有效时间（00–23时，00–59分） / Enter a valid calendar time")
		}
		switch d.Frequency {
		case "weekdays":
			return fmt.Sprintf("%d %d * * 1-5", minute, hour), nil
		case "weekly":
			day, err := strconv.Atoi(strings.TrimSpace(d.Weekday))
			if err != nil || day < 0 || day > 6 {
				return "", fmt.Errorf("请选择星期（0=周日，6=周六） / Select weekday 0–6")
			}
			return fmt.Sprintf("%d %d * * %d", minute, hour, day), nil
		case "monthly":
			day, err := strconv.Atoi(strings.TrimSpace(d.MonthDay))
			if err != nil || day < 1 || day > 31 {
				return "", fmt.Errorf("每月日期须为 1–31 / Day of month must be 1–31")
			}
			return fmt.Sprintf("%d %d %d * *", minute, hour, day), nil
		}
	}
	return "", fmt.Errorf("请选择支持的任务类型 / Select a supported frequency")
}
