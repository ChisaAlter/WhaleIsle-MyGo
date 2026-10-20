package dshbot

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"whaleisle/internal/store/storelocale"
)

// Match the original schedule.js Host grammar, including 1–525600 minutes.
// This builds a schedule, never a local estimate of nextRunAt.
var routineCreateDurationPattern = regexp.MustCompile(`^(\d+)([mhd])$`)

func routineCreateDuration(amount, unit string) (string, error) {
	raw := strings.TrimSpace(amount) + unit
	if len(raw) > 200 || !routineCreateDurationPattern.MatchString(raw) {
		return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotDurationMustBeAPositiveIntegerWith))
	}
	n, err := strconv.ParseUint(strings.TrimSpace(amount), 10, 64)
	multiplier := map[string]uint64{"m": 1, "h": 60, "d": 1440}[unit]
	if err != nil || multiplier == 0 || n < 1 || n > 525600/multiplier {
		return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotDurationMustBe1525600Minutes))
	}
	return strconv.FormatUint(n, 10) + unit, nil
}

func routineCreateSchedule(d RoutineCreateDraft) (string, error) {
	switch d.Frequency {
	case "", "daily":
		hour, hErr := strconv.Atoi(strings.TrimSpace(d.Hour))
		minute, mErr := strconv.Atoi(strings.TrimSpace(d.Minute))
		if hErr != nil || mErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotEnterAValidDailyTime))
		}
		return fmt.Sprintf("%d %d * * *", minute, hour), nil
	case "once":
		raw := strings.ToLower(strings.TrimSpace(d.OnceDelay))
		match := routineCreateDurationPattern.FindStringSubmatch(raw)
		if len(raw) > 200 || len(match) != 3 {
			return "", errors.New(storelocale.T(storelocale.KeyStoreDshbotEnterAOneShotDelaySuchAs30m1hOr))
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
