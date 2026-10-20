package schedule

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The original task-cron.ts dialect is numeric five-field cron. Starred day
// fields use intersection; two non-starred day fields use Vixie union.
// Host remains responsible for occurrence selection and dispatch.
type taskCronField struct {
	values        []int
	starred, full bool
}
type taskCron [5]taskCronField
type cronShape struct {
	kind               string
	step, hour, minute int
	days               []int
}

func parseTaskCron(expression string) (taskCron, error) {
	var result taskCron
	fields := strings.Fields(expression)
	if len(fields) != 5 || strings.TrimSpace(expression) != expression {
		return result, fmt.Errorf("Cron needs five fields without surrounding spaces")
	}
	bounds := [][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 7}}
	for i, raw := range fields {
		low, high := bounds[i][0], bounds[i][1]
		seen := map[int]bool{}
		for _, part := range strings.Split(raw, ",") {
			step := 1
			pieces := strings.Split(part, "/")
			if len(pieces) > 2 {
				return result, fmt.Errorf("invalid Cron field: %s", raw)
			}
			if len(pieces) == 2 {
				var err error
				step, err = cronNumber(pieces[1])
				if err != nil || step < 1 {
					return result, fmt.Errorf("invalid Cron step: %s", raw)
				}
			}
			base := pieces[0]
			start, end := low, high
			if base != "*" {
				nums := strings.Split(base, "-")
				if len(nums) > 2 || (len(nums) == 1 && len(pieces) == 2) {
					return result, fmt.Errorf("invalid Cron range: %s", raw)
				}
				var err error
				start, err = cronNumber(nums[0])
				if err != nil {
					return result, err
				}
				end = start
				if len(nums) == 2 {
					end, err = cronNumber(nums[1])
					if err != nil {
						return result, err
					}
				}
			}
			if start < low || end > high || start > end {
				return result, fmt.Errorf("Cron field outside %d–%d: %s", low, high, raw)
			}
			for v := start; v <= end; {
				n := v
				if i == 4 && n == 7 {
					n = 0
				}
				seen[n] = true
				if step > end-v {
					break
				}
				v += step
			}
		}
		for n := range seen {
			result[i].values = append(result[i].values, n)
		}
		sort.Ints(result[i].values)
		count := high - low + 1
		if i == 4 {
			count = 7
		}
		result[i].starred = strings.HasPrefix(raw, "*")
		result[i].full = len(seen) == count
	}
	return result, nil
}
func cronNumber(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty Cron value")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("Cron requires numeric values: %s", s)
		}
	}
	n, err := strconv.Atoi(s)
	if err == nil && int64(n) > 9007199254740991 {
		return 0, fmt.Errorf("Cron value exceeds the supported integer range")
	}
	return n, err
}
func (f taskCronField) unrestricted() bool { return f.starred && f.full }
func cronFullStep(f taskCronField, max int) int {
	if f.unrestricted() {
		return 1
	}
	v := f.values
	if len(v) < 2 || v[0] != 0 {
		return 0
	}
	step := v[1] - v[0]
	for i := 1; i < len(v); i++ {
		if v[i]-v[i-1] != step {
			return 0
		}
	}
	if v[len(v)-1]+step <= max {
		return 0
	}
	return step
}
func (p taskCron) shape() (cronShape, bool) {
	if !p[3].full {
		return cronShape{}, false
	}
	s := cronShape{step: 1}
	if len(p[0].values) == 1 {
		s.minute = p[0].values[0]
	}
	if len(p[1].values) == 1 {
		s.hour = p[1].values[0]
	}
	if p[4].unrestricted() {
		if p[2].unrestricted() {
			if step := cronFullStep(p[0], 59); step > 0 && p[1].full {
				s.kind = "minutely"
				s.step = step
				return s, true
			}
			if step := cronFullStep(p[1], 23); len(p[0].values) == 1 && step > 0 {
				s.kind = "hourly"
				s.step = step
				return s, true
			}
			if len(p[0].values) == 1 && len(p[1].values) == 1 {
				s.kind = "daily"
				return s, true
			}
		} else if len(p[0].values) == 1 && len(p[1].values) == 1 {
			s.kind = "monthly"
			s.days = append([]int(nil), p[2].values...)
			return s, true
		}
	} else if p[2].unrestricted() && len(p[0].values) == 1 && len(p[1].values) == 1 {
		s.kind = "weekly"
		for _, n := range p[4].values {
			if n == 0 {
				n = 7
			}
			s.days = append(s.days, n)
		}
		sort.Ints(s.days)
		return s, true
	}
	return cronShape{}, false
}
func cronList(values []int) string {
	v := append([]int(nil), values...)
	sort.Ints(v)
	parts := []string{}
	for i := 0; i < len(v); {
		j := i + 1
		for j < len(v) && v[j] == v[j-1]+1 {
			j++
		}
		if j-i >= 3 {
			parts = append(parts, fmt.Sprintf("%d-%d", v[i], v[j-1]))
		} else {
			for _, n := range v[i:j] {
				parts = append(parts, strconv.Itoa(n))
			}
		}
		i = j
	}
	return strings.Join(parts, ",")
}
func (s cronShape) expression() string {
	switch s.kind {
	case "minutely":
		if s.step == 1 {
			return "* * * * *"
		}
		return fmt.Sprintf("*/%d * * * *", s.step)
	case "hourly":
		if s.step == 1 {
			return fmt.Sprintf("%d * * * *", s.minute)
		}
		return fmt.Sprintf("%d */%d * * *", s.minute, s.step)
	case "daily":
		return fmt.Sprintf("%d %d * * *", s.minute, s.hour)
	case "weekly":
		days := append([]int(nil), s.days...)
		for i, n := range days {
			days[i] = n % 7
		}
		return fmt.Sprintf("%d %d * * %s", s.minute, s.hour, cronList(days))
	case "monthly":
		return fmt.Sprintf("%d %d %s * *", s.minute, s.hour, cronList(s.days))
	}
	return ""
}
func weekdayList(values []int) string {
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	parts := []string{}
	for _, n := range values {
		parts = append(parts, names[n%7])
	}
	return strings.Join(parts, ", ")
}
func (p taskCron) preview() string {
	day := "Every day"
	dom, dow := p[2], p[4]
	if dom.starred || dow.starred {
		if !dom.full && !dow.full {
			day = "Dates " + cronList(dom.values) + " AND " + weekdayList(dow.values)
		} else if !dom.full {
			day = "Dates " + cronList(dom.values)
		} else if !dow.full {
			day = weekdayList(dow.values)
		}
	} else if !dom.full && !dow.full {
		day = "Dates " + cronList(dom.values) + " OR " + weekdayList(dow.values)
	}
	clock := "hours " + cronList(p[1].values) + ", minutes " + cronList(p[0].values)
	if p[1].full {
		if step := cronFullStep(p[0], 59); step > 0 {
			clock = fmt.Sprintf("every %d minute(s)", step)
		} else {
			clock = "every hour at minute " + cronList(p[0].values)
		}
	} else if len(p[0].values)*len(p[1].values) <= 6 {
		times := []string{}
		for _, h := range p[1].values {
			for _, m := range p[0].values {
				times = append(times, fmt.Sprintf("%02d:%02d", h, m))
			}
		}
		clock = "at " + strings.Join(times, ", ")
	}
	if !p[3].full {
		day += " in months " + cronList(p[3].values)
	}
	return day + ", " + clock
}
func toggleNumber(values []int, n int) []int {
	result := append([]int(nil), values...)
	for i, v := range result {
		if v == n {
			if len(result) == 1 {
				return result
			}
			return append(result[:i], result[i+1:]...)
		}
	}
	result = append(result, n)
	sort.Ints(result)
	return result
}

// matches reports whether the expression fires at t, in the wall clock t already
// carries. Starred day fields intersect, two explicit day fields union, which is
// the same rule the preview text describes.
func (p taskCron) matches(t time.Time) bool {
	if !slices.Contains(p[0].values, t.Minute()) || !slices.Contains(p[1].values, t.Hour()) || !slices.Contains(p[3].values, int(t.Month())) {
		return false
	}
	dom, dow := p[2], p[4]
	domOK := slices.Contains(dom.values, t.Day())
	dowOK := slices.Contains(dow.values, int(t.Weekday()))
	switch {
	case dom.full && dow.full:
		return true
	case dom.full:
		return dowOK
	case dow.full:
		return domOK
	default:
		return domOK || dowOK
	}
}

// nextAfter is the first occurrence after from, looked for over six weeks so the
// monthly shapes still land inside the window. The host still owns dispatch; this
// only fills the form's next-run preview.
func (p taskCron) nextAfter(from time.Time, loc *time.Location) (time.Time, bool) {
	if loc == nil {
		loc = time.UTC
	}
	t := from.In(loc).Truncate(time.Minute).Add(time.Minute)
	limit := t.AddDate(0, 0, 45)
	for t.Before(limit) {
		if p.matches(t) {
			return t, true
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, false
}