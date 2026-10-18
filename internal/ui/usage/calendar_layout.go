package usage

import (
	"time"
	usagestore "whaleisle/internal/store/usage"
)

// Match the original month marker rule: use each week's first actual day;
// suppress a partial leading month when the next marker is less than two weeks away.
func calendarMonthLabels(start, end time.Time, days map[string]usagestore.DayUsage) map[string]string {
	type marker struct {
		week        int
		date, month string
	}
	var labels []marker
	previous := ""
	for week, index := start, 0; !week.After(end); week, index = week.AddDate(0, 0, 7), index+1 {
		month := ""
		for n := 0; n < 7; n++ {
			day := week.AddDate(0, 0, n).Format("2006-01-02")
			if _, ok := days[day]; ok {
				month = day[:7]
				break
			}
		}
		if month != "" && month != previous {
			labels = append(labels, marker{index, week.Format("2006-01-02"), month})
			previous = month
		}
	}
	if len(labels) > 1 && labels[1].week-labels[0].week < 2 {
		labels = labels[1:]
	}
	out := map[string]string{}
	for _, label := range labels {
		out[label.date] = label.month
	}
	return out
}
