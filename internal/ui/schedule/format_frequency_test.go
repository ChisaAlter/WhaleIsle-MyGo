package schedule

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCatalogFrequencyEveryOnceAndWallClock(t *testing.T) {
	now := time.Date(2026, 10, 28, 3, 30, 0, 0, time.UTC)
	for _, test := range []struct{ raw, en, zh string }{
		{`{"kind":"after"}`, "Once", "单次"},
		{`{"kind":"at"}`, "Once", "单次"},
		{`{"kind":"every","everySeconds":1}`, "Every 1 second", "1秒一次"},
		{`{"kind":"every","everySeconds":60}`, "Every 1 minute", "1分钟一次"},
		{`{"kind":"every","everySeconds":3600}`, "Every 1 hour", "1小时一次"},
		{`{"kind":"every","everySeconds":86400}`, "Every 1 day", "1天一次"},
		{`{"kind":"every","everySeconds":172800}`, "Every 2 days", "2天一次"},
		{`{"kind":"every","everySeconds":90}`, "Every 90 seconds", "90秒一次"},
		{`{"kind":"daily","time":"09:00:00.123","timeZone":"UTC"}`, "Daily at 09:00 (UTC+00:00)", "每天 09:00（UTC+00:00）"},
		{`{"kind":"weekly","time":"09:01:02.003","weekdays":[1,3,7],"timeZone":"UTC"}`, "Weekly on Mon, Wed, Sun at 09:01:02 (UTC+00:00)", "每周一、三、日 09:01:02（UTC+00:00）"},
		{`{"kind":"daily","time":"11:12:00","timeZone":"Invalid/Original"}`, "Daily at 11:12 (Invalid/Original)", "每天 11:12（Invalid/Original）"},
		{`{"kind":"every","everySeconds":0}`, "", ""},
		{`{"kind":"unknown"}`, "", ""},
		{`{`, "", ""},
	} {
		for _, locale := range []string{"en", "zh-CN"} {
			want := test.en
			if locale == "zh-CN" {
				want = test.zh
			}
			if got := FormatCatalogFrequency(json.RawMessage(test.raw), now, locale); got != want {
				t.Errorf("%s %s: got%q want%q", locale, test.raw, got, want)
			}
		}
	}
}

func TestCatalogFrequencyCronDescriptionsMatchSourceDictionary(t *testing.T) {
	for _, test := range []struct{ expression, en, zh string }{
		{"* * * * *", "Every minute", "每分钟"},
		{"*/15 * * * *", "Every 15 minutes", "每 15 分钟"},
		{"0 * * * *", "Every hour", "每小时"},
		{"0 */2 * * *", "Every 2 hours", "每 2 小时"},
		{"5,20 * * * *", "Every day every hour at minute 5, 20", "每天 每小时的第 5、20 分钟"},
		{"0 9,15 * * 1-5", "Mon–Fri at 09:00, 15:00", "周一至周五 09:00、15:00"},
		{"0 9 1 * 1", "Day 1 of every month or Mon at 09:00", "每月 1 日或周一 09:00"},
		{"0 9 */15 * 1", "Day 1, 16, 31 of every month and Mon at 09:00", "每月 1、16、31 日且周一 09:00"},
		{"0 9 * 1,7 *", "Every day in January, July at 09:00", "每天（一月、七月） 09:00"},
		{"*/15 9-17 * * 1-5", "Mon–Fri every 15 minutes during hours 09–17", "周一至周五 09 至 17 点内每 15 分钟"},
		{"1,2,3,4 9,15 * * *", "Every day at minute 1, 2, 3, 4 of hours 09, 15", "每天 09、15 点的第 1、2、3、4 分钟"},
		{"* 9,15 * * *", "Every day every minute during hours 09, 15", "每天 09、15 点的每分钟"},
		{"0 9 * * 0,7", "Sun at 09:00", "周日 09:00"},
		{"@daily", "Cron @daily", "Cron @daily"},
		{"0 9 * * MON", "Cron 0 9 * * MON", "Cron 0 9 * * MON"},
	} {
		for _, locale := range []string{"en", "zh-CN"} {
			var got string
			parsed, err := parseTaskCron(test.expression)
			if err == nil {
				got = catalogCronPreview(parsed, locale)
			} else {
				got = "Cron " + test.expression
			}
			want := test.en
			if locale == "zh-CN" {
				want = test.zh
			}
			if got != want {
				t.Errorf("%s %s: got%q want%q", locale, test.expression, got, want)
			}
		}
	}
}

func TestCatalogZonePermanentPinnedICUGoldens(t *testing.T) {
	for _, test := range []struct{ zone, locale, instant, want string }{
		{"America/New_York", "en", "2026-01-15T12:00:00Z", "UTC-05:00 · Eastern Time"},
		{"America/New_York", "en", "2026-07-15T12:00:00Z", "UTC-04:00 · Eastern Time"},
		{"US/Eastern", "zh-CN", "2026-01-15T12:00:00Z", "UTC-05:00 · 北美东部时间"},
		{"America/New_York", "zh-CN", "2026-07-15T12:00:00Z", "UTC-04:00 · 北美东部时间"},
		{"Asia/Shanghai", "zh-CN", "2026-10-28T03:30:00Z", "UTC+08:00 · 中国标准时间"},
		{"Asia/Kathmandu", "en", "2026-10-28T03:30:00Z", "UTC+05:45 · Nepal Time"},
		{"Europe/London", "en", "2026-07-15T12:00:00Z", "UTC+01:00 · United Kingdom Time"},
		{"Europe/Dublin", "en", "2026-07-15T12:00:00Z", "UTC+01:00 · Ireland Time"},
		{"Africa/Casablanca", "en", "2026-07-15T12:00:00Z", "UTC+01:00 · Morocco Time"},
		{"Africa/Casablanca", "en", "2026-10-28T03:30:00Z", "UTC+00:00 · Western European Time"},
		{"Africa/Casablanca", "en", "2027-01-15T12:00:00Z", "UTC+00:00 · Western European Time"},
		{"Africa/Casablanca", "en", "2027-07-15T12:00:00Z", "UTC+00:00 · Western European Standard Time"},
		{"Africa/El_Aaiun", "zh-CN", "2026-07-15T12:00:00Z", "UTC+01:00 · 西撒哈拉时间"},
		{"Africa/El_Aaiun", "zh-CN", "2026-10-28T03:30:00Z", "UTC+00:00 · 西欧时间"},
		{"Etc/GMT+8", "en", "2026-10-28T03:30:00Z", "UTC-08:00"},
		{"UTC", "zh-CN", "2026-10-28T03:30:00Z", "UTC+00:00"},
	} {
		at, err := time.Parse(time.RFC3339, test.instant)
		if err != nil {
			t.Fatal(err)
		}
		if got := formatCatalogZone(test.zone, at, test.locale); got != test.want {
			t.Errorf("%s %s %s got%q want%q", test.zone, test.locale, test.instant, got, test.want)
		}
	}
}

// A development-only oracle is optional in ordinary repository runs. It was
// generated from this project's independent Intl script, not installed code;
// the fixed corpus includes every ICU inventory zone and accepted CLDR alias,
// both supported languages, and winter/summer/current instants. The application
// never executes JavaScript or loads this test fixture at run time.
func TestCatalogZoneMatchesPinnedICUCorpus(t *testing.T) {
	path := os.Getenv("WHALEISLE_ZONE_ORACLE")
	if path == "" {
		t.Skip("set WHALEISLE_ZONE_ORACLE to the independently generated ICU48 corpus")
	}
	bytes, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Versions struct {
			CLDR string `json:"cldr"`
		} `json:"versions"`
		Rows []struct{ Zone, Locale, At, Expected string } `json:"rows"`
	}
	if err := json.Unmarshal(bytes, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Versions.CLDR != "48.0" || len(corpus.Rows) < 2500 {
		t.Fatalf("wrong oracle version or incomplete corpus: %+v rows%d", corpus.Versions, len(corpus.Rows))
	}
	differences := []string{}
	for _, row := range corpus.Rows {
		at, err := time.Parse(time.RFC3339, row.At)
		if err != nil {
			t.Fatal(err)
		}
		if got := formatCatalogZone(row.Zone, at, row.Locale); got != row.Expected {
			differences = append(differences, row.Zone+" "+row.Locale+" "+row.At+" got="+got+" want="+row.Expected)
		}
	}
	if len(differences) > 0 {
		output := strings.Join(differences, "\n")
		if path := os.Getenv("WHALEISLE_ZONE_DIFF"); path != "" {
			if err := os.WriteFile(path, []byte(output), 0600); err != nil {
				t.Fatal(err)
			}
		}
		for _, difference := range differences[:min(30, len(differences))] {
			t.Log(difference)
		}
		t.Fatalf("%d/%d zone labels differ from independently pinned ICU48", len(differences), len(corpus.Rows))
	}
	t.Logf("all %d ICU48 timezone/alias/language/instant cases match", len(corpus.Rows))
}
