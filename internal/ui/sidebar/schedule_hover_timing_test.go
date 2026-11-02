package sidebar

import (
	"encoding/json"
	"testing"
	"time"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/i18n"
)

func TestScheduleHoverTimingAbsoluteDeviceZoneAndLocale(t *testing.T) {
	oldZone, oldLocale := time.Local, i18n.Locale()
	t.Cleanup(func() { time.Local = oldZone; i18n.SetLocale(oldLocale) })
	time.Local = time.FixedZone("device UTC+08", 8*60*60)
	at := time.Date(2026, 9, 19, 7, 51, 44, 0, time.UTC)
	row := schedstore.Entry{ScheduledAt: at.Format(time.RFC3339Nano), Raw: json.RawMessage(`{"kind":"daily","timeZone":"America/New_York"}`)}
	for _, tc := range []struct{ locale, want string }{
		{i18n.LocaleEN, "Sep 19, 2026, 15:51 (Due now)"},
		{i18n.LocaleZH, "9月19日 15:51 (现在到期)"},
	} {
		i18n.SetLocale(tc.locale)
		if got := schedulePreviewTiming(row, at); got != tc.want {
			t.Errorf("locale=%s got=%q want=%q", tc.locale, got, tc.want)
		}
		absolute, relative := schedulePreviewParts(row, at)
		if absolute+" "+relative != tc.want || relative != workspaceText("(现在到期)", "(Due now)") {
			t.Errorf("parts are not independently renderable: %q / %q", absolute, relative)
		}
	}
	i18n.SetLocale(i18n.LocaleEN)
	row.ScheduledAt = "2026-12-31T16:00:00Z"
	if got := schedulePreviewTiming(row, time.Date(2026, 12, 31, 16, 0, 0, 0, time.UTC)); got != "Jan 1, 2027, 00:00 (Due now)" {
		t.Fatal("device year rollover or midnight hour is wrong", got)
	}
	i18n.SetLocale(i18n.LocaleZH)
	if got := schedulePreviewTiming(row, time.Date(2026, 12, 31, 16, 0, 0, 0, time.UTC)); got != "1月1日 00:00 (现在到期)" {
		t.Fatal(got)
	}
}

func TestScheduleHoverTimingInvalidInstantPreservesRaw(t *testing.T) {
	oldLocale := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(oldLocale) })
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleZH} {
		i18n.SetLocale(locale)
		row := schedstore.Entry{ScheduledAt: "invalid-original-instant"}
		absolute, relative := schedulePreviewParts(row, time.Time{})
		if absolute != row.ScheduledAt || relative != workspaceText("(下次运行时间不可用)", "(Next run unavailable)") {
			t.Fatalf("invalid instant fabricated a stamp: %q / %q", absolute, relative)
		}
	}
}

func TestScheduleHoverTimingRelativeRoundingAndStableSample(t *testing.T) {
	oldZone, oldLocale := time.Local, i18n.Locale()
	t.Cleanup(func() { time.Local = oldZone; i18n.SetLocale(oldLocale) })
	time.Local = time.UTC
	i18n.SetLocale(i18n.LocaleEN)
	now := time.Date(2026, 9, 19, 15, 51, 0, 0, time.UTC)
	for _, tc := range []struct {
		delta    time.Duration
		relative string
	}{
		{0, "Due now"}, {time.Nanosecond, "in 1 second"}, {-time.Nanosecond, "1 second overdue"},
		{59*time.Second + time.Millisecond, "in 60 seconds"}, {-(59*time.Second + time.Millisecond), "59 seconds overdue"},
		{60 * time.Second, "in 1 minute"}, {-60 * time.Second, "1 minute overdue"},
		{3599 * time.Second, "in 60 minutes"}, {-3599 * time.Second, "59 minutes overdue"},
		{time.Hour, "in 1 hour"}, {-time.Hour, "1 hour overdue"},
		{24 * time.Hour, "in 1 day"}, {-24 * time.Hour, "1 day overdue"},
	} {
		row := schedstore.Entry{ScheduledAt: now.Add(tc.delta).Format(time.RFC3339Nano)}
		want := now.Add(tc.delta).Format("Jan 2, 2006, 15:04") + " (" + tc.relative + ")"
		if got := schedulePreviewTiming(row, now); got != want {
			t.Errorf("delta=%s got=%q want=%q", tc.delta, got, want)
		}
		if first, again := schedulePreviewTiming(row, now), schedulePreviewTiming(row, now); first != again {
			t.Fatal("fixed hover sample changed between renders")
		}
	}
}
