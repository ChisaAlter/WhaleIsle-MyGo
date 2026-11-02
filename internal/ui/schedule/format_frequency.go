package schedule

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatCatalogFrequency describes the stored rule using the schedule.catalog
// dictionary. It never computes an occurrence or changes the Host's rule. The
// supplied instant is also used for zone names and offsets, keeping a mounted
// hover card stable. Unlike the manager's host-zone view, catalog hover cards
// always show a wall-clock rule's zone, including the viewer's own zone.
func FormatCatalogFrequency(raw json.RawMessage, now time.Time, locale string) string {
	var rule struct {
		Kind, Time, TimeZone, Expression string
		EverySeconds                     int64
		Weekdays                         []int
	}
	if json.Unmarshal(raw, &rule) != nil {
		return ""
	}
	switch rule.Kind {
	case "after", "at":
		return catalogCopy(locale, "单次", "Once")
	case "every":
		if rule.EverySeconds <= 0 {
			return ""
		}
		for _, unit := range []struct {
			seconds int64
			zh, en  string
		}{{86400, "天", "day"}, {3600, "小时", "hour"}, {60, "分钟", "minute"}, {1, "秒", "second"}} {
			if rule.EverySeconds%unit.seconds != 0 {
				continue
			}
			value := rule.EverySeconds / unit.seconds
			english := unit.en
			if value != 1 {
				english += "s"
			}
			return fmt.Sprintf(catalogCopy(locale, "%d%s一次", "Every %d %s"), value, catalogCopy(locale, unit.zh, english))
		}
	case "daily":
		return catalogFrequencyZone(fmt.Sprintf(catalogCopy(locale, "每天 %s", "Daily at %s"), catalogClock(rule.Time)), rule.TimeZone, now, locale)
	case "weekly":
		days := make([]string, 0, len(rule.Weekdays))
		for _, day := range rule.Weekdays {
			days = append(days, catalogWeekday(day, locale))
		}
		text := fmt.Sprintf(catalogCopy(locale, "每周%s %s", "Weekly on %s at %s"), strings.Join(days, catalogListJoin(locale)), catalogClock(rule.Time))
		return catalogFrequencyZone(text, rule.TimeZone, now, locale)
	case "cron":
		text := "Cron " + rule.Expression
		if parsed, err := parseTaskCron(rule.Expression); err == nil {
			text = catalogCronPreview(parsed, locale)
		}
		return catalogFrequencyZone(text, rule.TimeZone, now, locale)
	}
	return ""
}

func catalogCopy(locale, zh, en string) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		return zh
	}
	return en
}

func catalogListJoin(locale string) string { return catalogCopy(locale, "、", ", ") }

func catalogClock(clock string) string {
	// JS clockLabel removes only a trailing decimal fraction and then one
	// trailing zero-seconds component; it does not truncate nonzero seconds.
	if dot := strings.LastIndexByte(clock, '.'); dot >= 0 && dot+1 < len(clock) {
		digits := true
		for _, ch := range clock[dot+1:] {
			if ch < '0' || ch > '9' {
				digits = false
				break
			}
		}
		if digits {
			clock = clock[:dot]
		}
	}
	return strings.TrimSuffix(clock, ":00")
}

func catalogFrequencyZone(rule, zone string, now time.Time, locale string) string {
	label := formatCatalogZone(zone, now, locale)
	return rule + fmt.Sprintf(catalogCopy(locale, "（%s）", " (%s)"), label)
}

func catalogWeekday(iso int, locale string) string {
	if iso < 1 || iso > 7 {
		iso = 1
	} // Matches the source dictionary-key fallback.
	zh := [...]string{"一", "二", "三", "四", "五", "六", "日"}
	en := [...]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	return catalogCopy(locale, zh[iso-1], en[iso-1])
}

func catalogNumbers(values []int, locale string, padded bool) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		part := strconv.Itoa(value)
		if padded {
			part = fmt.Sprintf("%02d", value)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, catalogListJoin(locale))
}

func catalogCronWeekdays(values []int, locale string) string {
	iso := []int{}
	for _, value := range values {
		if value != 0 {
			iso = append(iso, value)
		}
	}
	for _, value := range values {
		if value == 0 {
			iso = append(iso, 7)
		}
	}
	name := func(value int) string { return catalogCopy(locale, "周", "") + catalogWeekday(value, locale) }
	parts := []string{}
	for i := 0; i < len(iso); {
		j := i + 1
		for j < len(iso) && iso[j] == iso[j-1]+1 {
			j++
		}
		if j-i >= 3 {
			parts = append(parts, name(iso[i])+catalogCopy(locale, "至", "–")+name(iso[j-1]))
		} else {
			for _, value := range iso[i:j] {
				parts = append(parts, name(value))
			}
		}
		i = j
	}
	return strings.Join(parts, catalogListJoin(locale))
}

func catalogCronDay(p taskCron, locale string) (string, bool) {
	months := ""
	if !p[3].full {
		parts := []string{}
		zh := [...]string{"一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"}
		for _, month := range p[3].values {
			parts = append(parts, catalogCopy(locale, zh[month-1], time.Month(month).String()))
		}
		months = fmt.Sprintf(catalogCopy(locale, "（%s）", " in %s"), strings.Join(parts, catalogListJoin(locale)))
	}
	dom, dow := p[2], p[4]
	weekdays, days := catalogCronWeekdays(dow.values, locale), catalogNumbers(dom.values, locale, false)
	every := catalogCopy(locale, "每天", "Every day") + months
	if dom.starred || dow.starred {
		switch {
		case dom.full && dow.full:
			return every, months == ""
		case dom.full:
			return weekdays + months, false
		case dow.full:
			return fmt.Sprintf(catalogCopy(locale, "每月 %s 日%s", "Day %s of every month%s"), days, months), false
		default:
			return fmt.Sprintf(catalogCopy(locale, "每月 %s 日且%s%s", "Day %s of every month and %s%s"), days, weekdays, months), false
		}
	}
	if dom.full || dow.full {
		return every, months == ""
	}
	return fmt.Sprintf(catalogCopy(locale, "每月 %s 日或%s%s", "Day %s of every month or %s%s"), days, weekdays, months), false
}

func catalogCronHours(values []int, locale string) string {
	if len(values) >= 2 && values[len(values)-1]-values[0] == len(values)-1 {
		return fmt.Sprintf(catalogCopy(locale, "%02d 至 %02d", "%02d–%02d"), values[0], values[len(values)-1])
	}
	return catalogNumbers(values, locale, true)
}

func catalogCronTime(p taskCron, locale string) (text, joined string, interval bool) {
	minutes, hours := p[0], p[1]
	step := 0
	if !minutes.unrestricted() {
		step = cronFullStep(minutes, 59)
	}
	if hours.unrestricted() {
		switch {
		case minutes.unrestricted():
			return catalogCopy(locale, "每分钟", "Every minute"), catalogCopy(locale, "每分钟", "every minute"), true
		case step >= 2:
			return fmt.Sprintf(catalogCopy(locale, "每 %d 分钟", "Every %d minutes"), step), fmt.Sprintf(catalogCopy(locale, "每 %d 分钟", "every %d minutes"), step), true
		case len(minutes.values) == 1 && minutes.values[0] == 0:
			return catalogCopy(locale, "每小时", "Every hour"), catalogCopy(locale, "每小时", "every hour"), true
		default:
			values := catalogNumbers(minutes.values, locale, false)
			return fmt.Sprintf(catalogCopy(locale, "每小时的第 %s 分钟", "Every hour at minute %s"), values), fmt.Sprintf(catalogCopy(locale, "每小时的第 %s 分钟", "every hour at minute %s"), values), false
		}
	}
	hourText := catalogCronHours(hours.values, locale)
	if minutes.unrestricted() {
		text = fmt.Sprintf(catalogCopy(locale, "%s 点的每分钟", "every minute during hours %s"), hourText)
	} else if step >= 2 {
		if strings.HasPrefix(strings.ToLower(locale), "zh") {
			text = fmt.Sprintf("%s 点内每 %d 分钟", hourText, step)
		} else {
			text = fmt.Sprintf("every %d minutes during hours %s", step, hourText)
		}
	} else if hourStep := cronFullStep(hours, 23); len(minutes.values) == 1 && minutes.values[0] == 0 && hourStep >= 2 {
		return fmt.Sprintf(catalogCopy(locale, "每 %d 小时", "Every %d hours"), hourStep), fmt.Sprintf(catalogCopy(locale, "每 %d 小时", "every %d hours"), hourStep), true
	} else if len(minutes.values)*len(hours.values) <= 6 {
		parts := []string{}
		for _, hour := range hours.values {
			for _, minute := range minutes.values {
				parts = append(parts, fmt.Sprintf("%02d:%02d", hour, minute))
			}
		}
		text = catalogCopy(locale, "", "at ") + strings.Join(parts, catalogListJoin(locale))
	} else {
		if strings.HasPrefix(strings.ToLower(locale), "zh") {
			text = fmt.Sprintf("%s 点的第 %s 分钟", hourText, catalogNumbers(minutes.values, locale, false))
		} else {
			text = fmt.Sprintf("at minute %s of hours %s", catalogNumbers(minutes.values, locale, false), hourText)
		}
	}
	return text, text, false
}

func catalogCronPreview(parsed taskCron, locale string) string {
	day, every := catalogCronDay(parsed, locale)
	text, joined, interval := catalogCronTime(parsed, locale)
	if every && interval {
		return text
	}
	return day + " " + joined
}
