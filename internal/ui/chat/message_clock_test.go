package chat

import (
	"testing"
	"time"

	"whaleisle/internal/ui/i18n"
)

func TestMessageClockCalendarAndLocale(t *testing.T) {
	before := i18n.Locale()
	defer i18n.SetLocale(before)
	zone := time.FixedZone("test-zone", 8*3600)
	now := time.Date(2026, 10, 23, 15, 0, 0, 0, zone)
	for _, tc := range []struct {
		locale string
		day    time.Time
		want   string
	}{
		{i18n.LocaleEN, time.Date(2026, 10, 23, 9, 4, 0, 0, zone), "09:04"},
		{i18n.LocaleEN, time.Date(2026, 10, 22, 9, 4, 0, 0, zone), "10/22 09:04"},
		{i18n.LocaleEN, time.Date(2025, 1, 2, 9, 4, 0, 0, zone), "2025-1-2 09:04"},
		{i18n.LocaleZH, time.Date(2026, 10, 22, 9, 4, 0, 0, zone), "10月22日 09:04"},
		{i18n.LocaleZH, time.Date(2025, 1, 2, 9, 4, 0, 0, zone), "2025年1月2日 09:04"},
		{i18n.LocaleEN, time.UnixMilli(0).In(zone), "1970-1-1 08:00"},
		{i18n.LocaleEN, time.UnixMilli(-1).In(zone), "1970-1-1 07:59"},
	} {
		i18n.SetLocale(tc.locale)
		if got := formatMessageClock(float64(tc.day.UnixMilli()), now); got != tc.want {
			t.Errorf("%v: %q want %q", tc.day, got, tc.want)
		}
	}
}
