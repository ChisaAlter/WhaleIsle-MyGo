package schedule

import (
	"testing"
	"time"
	_ "time/tzdata"
)

func TestWindowsScheduleZoneMapping(t *testing.T) {
	for key, want := range map[string]string{"China Standard Time": "Asia/Shanghai", "Pacific Standard Time": "America/Los_Angeles", "India Standard Time": "Asia/Calcutta", "GMT Standard Time": "Europe/London", "UTC": "Etc/UTC", "not a Windows zone": "", "中国标准时间": "", "PST": ""} {
		if got := windowsScheduleZone(key); got != want {
			t.Errorf("%q: got %q want %q", key, got, want)
		}
	}
	for key, zone := range windowsScheduleZones {
		if _, err := time.LoadLocation(zone); err != nil {
			t.Errorf("%s -> %s unavailable: %v", key, zone, err)
		}
	}
	loc, _ := time.LoadLocation(windowsScheduleZone("Pacific Standard Time"))
	_, winter := time.Date(2026, 1, 15, 12, 0, 0, 0, loc).Zone()
	_, summer := time.Date(2026, 7, 15, 12, 0, 0, 0, loc).Zone()
	if winter != -8*3600 || summer != -7*3600 {
		t.Fatal("mapping must retain DST rules, not a fixed offset")
	}
}
