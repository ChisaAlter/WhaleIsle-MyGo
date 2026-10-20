package dshbot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"whaleisle/internal/store/storelocale"
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
			return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotCronNeedsFiveFieldsAtMost200Chara))
		}
		for _, field := range fields {
			if len(utf16.Encode([]rune(field))) > 40 {
				return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotEachCronFieldMayContainAtMost40C))
			}
		}
		// Preserve the exact trimmed expression rather than guessing parser semantics.
		return raw, nil
	case "weekdays", "weekly", "monthly":
		hour, hErr := strconv.Atoi(strings.TrimSpace(d.Hour))
		minute, mErr := strconv.Atoi(strings.TrimSpace(d.Minute))
		if hErr != nil || mErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotEnterAValidCalendarTime))
		}
		switch d.Frequency {
		case "weekdays":
			return fmt.Sprintf("%d %d * * 1-5", minute, hour), nil
		case "weekly":
			day, err := strconv.Atoi(strings.TrimSpace(d.Weekday))
			if err != nil || day < 0 || day > 6 {
				return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotSelectWeekday06))
			}
			return fmt.Sprintf("%d %d * * %d", minute, hour, day), nil
		case "monthly":
			day, err := strconv.Atoi(strings.TrimSpace(d.MonthDay))
			if err != nil || day < 1 || day > 31 {
				return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotDayOfMonthMustBe131))
			}
			return fmt.Sprintf("%d %d %d * *", minute, hour, day), nil
		}
	}
	return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotSelectASupportedFrequency))
}
