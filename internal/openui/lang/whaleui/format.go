package whaleui

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/openui/lang"
)

func now(e env) time.Time {
	if e != nil && e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func loc(e env) *time.Location {
	if e != nil && e.Location != nil {
		return e.Location
	}
	return time.Local
}

func locale(e env) string {
	if e != nil && strings.HasPrefix(strings.ToLower(e.Locale), "zh") {
		return "zh"
	}
	return "en"
}

func isoDate(t time.Time) string {
	return pad(t.Year(), 4) + "-" + pad(int(t.Month()), 2) + "-" + pad(t.Day(), 2)
}

func pad(n, w int) string {
	s := strconv.Itoa(n)
	for len(s) < w {
		s = "0" + s
	}
	return s
}

var (
	dateOnly = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	isoFull  = regexp.MustCompile(`^(\d{4})(?:-(\d{2})(?:-(\d{2}))?)?(?:T(\d{2}):(\d{2})(?::(\d{2})(?:\.(\d{1,9}))?)?)?(Z|[+-]\d{2}:\d{2})?$`)
)

var errInvalidDate = errors.New("Invalid date")

// toDate is the viewer's `tc`: "YYYY-MM-DD" is a local calendar date, a
// number is a millisecond timestamp, other strings use ISO 8601 date-time
// parsing (date-only forms are UTC, date-times without offset are local).
func toDate(v lang.Value, l *time.Location) (time.Time, error) {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) || math.Abs(t) > 8.64e15 {
			return time.Time{}, errInvalidDate
		}
		return time.UnixMilli(int64(t)).In(l), nil
	case string:
		if m := dateOnly.FindStringSubmatch(t); m != nil {
			y, _ := strconv.Atoi(m[1])
			mo, _ := strconv.Atoi(m[2])
			d, _ := strconv.Atoi(m[3])
			// JS Date normalizes overflow (2026-02-30 → Mar 2).
			return time.Date(y, time.Month(mo), d, 0, 0, 0, 0, l), nil
		}
		m := isoFull.FindStringSubmatch(t)
		if m == nil {
			return time.Time{}, errInvalidDate
		}
		atoi := func(s string, d int) int {
			if s == "" {
				return d
			}
			n, _ := strconv.Atoi(s)
			return n
		}
		y, mo, d := atoi(m[1], 0), atoi(m[2], 1), atoi(m[3], 1)
		if mo < 1 || mo > 12 || d < 1 || d > 31 {
			return time.Time{}, errInvalidDate
		}
		hh, mm, ss := atoi(m[4], 0), atoi(m[5], 0), atoi(m[6], 0)
		ns := 0
		if m[7] != "" {
			frac := (m[7] + "000000000")[:9]
			ns, _ = strconv.Atoi(frac)
		}
		zone := l
		switch {
		case m[8] == "Z" || m[8] == "" && m[4] == "":
			zone = time.UTC
		case m[8] != "":
			sign := 1
			if m[8][0] == '-' {
				sign = -1
			}
			oh, _ := strconv.Atoi(m[8][1:3])
			om, _ := strconv.Atoi(m[8][4:6])
			zone = time.FixedZone("", sign*(oh*3600+om*60))
		}
		return time.Date(y, time.Month(mo), d, hh, mm, ss, ns, zone).In(l), nil
	}
	return time.Time{}, errInvalidDate
}

// addMonths is the viewer's `qoe`: months later, day clamped to the
// destination month.
func addMonths(v lang.Value, months float64, l *time.Location) (time.Time, error) {
	t, err := toDate(v, l)
	if err != nil {
		return t, err
	}
	day := t.Day()
	first := time.Date(t.Year(), t.Month(), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), l).AddDate(0, int(math.Trunc(months)), 0)
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, l).Day()
	if day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), l), nil
}

// startOf is the viewer's `Voe` (Monday-based weeks).
func startOf(v lang.Value, unit string, l *time.Location) (time.Time, error) {
	t, err := toDate(v, l)
	if err != nil {
		return t, err
	}
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, l)
	switch unit {
	case "week":
		t = t.AddDate(0, 0, -((int(t.Weekday()) + 6) % 7))
	case "month":
		t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, l)
	case "year":
		t = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, l)
	}
	return t, nil
}

func utcDays(t time.Time) float64 {
	return float64(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()) / 86400
}

var (
	monthsShort = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	weekdaysZh  = []string{"日", "一", "二", "三", "四", "五", "六"}
)

// formatDate approximates Intl.DateTimeFormat for en-US and zh-CN.
func formatDate(t time.Time, style, lc string) string {
	if lc == "zh" {
		y, m, d := strconv.Itoa(t.Year()), strconv.Itoa(int(t.Month())), strconv.Itoa(t.Day())
		hm := pad(t.Hour(), 2) + ":" + pad(t.Minute(), 2)
		switch style {
		case "short":
			return m + "月" + d + "日"
		case "long":
			return y + "年" + m + "月" + d + "日星期" + weekdaysZh[t.Weekday()]
		case "weekday":
			return "周" + weekdaysZh[t.Weekday()]
		case "month":
			return y + "年" + m + "月"
		case "time":
			return hm
		case "datetime":
			return y + "/" + m + "/" + d + " " + hm
		}
		return y + "年" + m + "月" + d + "日"
	}
	hour := t.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	ampm := "AM"
	if t.Hour() >= 12 {
		ampm = "PM"
	}
	clock := strconv.Itoa(hour) + ":" + pad(t.Minute(), 2) + " " + ampm
	date := monthsShort[t.Month()-1] + " " + strconv.Itoa(t.Day()) + ", " + strconv.Itoa(t.Year())
	switch style {
	case "short":
		return monthsShort[t.Month()-1] + " " + strconv.Itoa(t.Day())
	case "long":
		return t.Weekday().String() + ", " + t.Month().String() + " " + strconv.Itoa(t.Day()) + ", " + strconv.Itoa(t.Year())
	case "weekday":
		return t.Weekday().String()[:3]
	case "month":
		return monthsShort[t.Month()-1] + " " + strconv.Itoa(t.Year())
	case "time":
		return clock
	case "datetime":
		return date + ", " + clock
	}
	return date
}

var currencySymbols = map[string]map[string]string{
	"en": {"USD": "$", "EUR": "€", "GBP": "£", "JPY": "¥", "CNY": "CN¥", "INR": "₹", "KRW": "₩", "CAD": "CA$", "AUD": "A$", "HKD": "HK$", "TWD": "NT$"},
	"zh": {"USD": "US$", "EUR": "€", "GBP": "£", "JPY": "JP¥", "CNY": "¥", "INR": "₹", "KRW": "￦", "CAD": "CA$", "AUD": "AU$", "HKD": "HK$", "TWD": "NT$"},
}

var zeroDecimalCurrencies = map[string]bool{"JPY": true, "KRW": true}

// formatNumber approximates Intl.NumberFormat for en-US and zh-CN: number
// (max 3 fraction digits), currency, percent (45 means 45%) and compact.
func formatNumber(v float64, kind string, decimals *int, currency, lc string) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	cur := strings.ToUpper(currency)
	useCurrency := kind == "currency" || kind == "compact" && currency != ""
	if useCurrency && cur == "" {
		cur = "USD"
	}
	minF, maxF := 0, 3
	if useCurrency {
		minF, maxF = 2, 2
		if zeroDecimalCurrencies[cur] {
			minF, maxF = 0, 0
		}
	}
	if kind == "percent" {
		minF, maxF = 0, 0
	}
	suffix := ""
	if kind == "compact" {
		v, suffix = compactScale(v, lc)
		if decimals == nil {
			// ICU compact notation keeps two significant digits for small
			// scaled values: 1.2K, 12K, 123K.
			minF, maxF = 0, 0
			if math.Abs(v) < 10 {
				maxF = 1
			}
		}
	}
	if decimals != nil {
		minF, maxF = *decimals, *decimals
	}
	if math.IsInf(v, 0) {
		s := "∞"
		if v < 0 {
			s = "-∞"
		}
		return s
	}
	neg := v < 0 || v == 0 && math.Signbit(v)
	body := groupDigits(fixed(math.Abs(v), minF, maxF))
	if neg && strings.Trim(body, "0.,") == "" {
		neg = false
	}
	out := body + suffix
	if kind == "percent" {
		out += "%"
	}
	if useCurrency {
		sym, ok := currencySymbols[lc][cur]
		if !ok {
			sym = cur + "\u00a0"
		}
		out = sym + out
	}
	if neg {
		out = "-" + out
	}
	return out
}

func compactScale(v float64, lc string) (float64, string) {
	a := math.Abs(v)
	if lc == "zh" {
		switch {
		case a >= 1e12:
			return v / 1e12, "万亿"
		case a >= 1e8:
			return v / 1e8, "亿"
		case a >= 1e4:
			return v / 1e4, "万"
		}
		return v, ""
	}
	switch {
	case a >= 1e12:
		return v / 1e12, "T"
	case a >= 1e9:
		return v / 1e9, "B"
	case a >= 1e6:
		return v / 1e6, "M"
	case a >= 1e3:
		return v / 1e3, "K"
	}
	return v, ""
}

// fixed rounds half away from zero (ICU half-even differs only on exact
// binary ties) and trims fraction zeros beyond minF.
func fixed(v float64, minF, maxF int) string {
	s := strconv.FormatFloat(v, 'f', maxF, 64)
	if dot := strings.IndexByte(s, '.'); dot >= 0 {
		end := len(s)
		for end > dot+1+minF && s[end-1] == '0' {
			end--
		}
		if end == dot+1 {
			end = dot
		}
		s = s[:end]
	}
	return s
}

func groupDigits(s string) string {
	intPart, frac, hasFrac := strings.Cut(s, ".")
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if hasFrac {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	return b.String()
}
