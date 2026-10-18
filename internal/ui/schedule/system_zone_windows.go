//go:build windows

package schedule

import (
	"syscall"
	"unsafe"
)

// DYNAMIC_TIME_ZONE_INFORMATION: the stable key is not StandardName, which
// Windows localizes. Read-only API; no command or system setting is changed.
type scheduleDynamicTimezone struct {
	Bias                        int32
	StandardName                [32]uint16
	StandardDate                syscall.Systemtime
	StandardBias                int32
	DaylightName                [32]uint16
	DaylightDate                syscall.Systemtime
	DaylightBias                int32
	TimeZoneKeyName             [128]uint16
	DynamicDaylightTimeDisabled uint8
}

var getScheduleTimezone = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDynamicTimeZoneInformation")

func platformScheduleZone() string {
	if getScheduleTimezone.Find() != nil {
		return ""
	}
	var zone scheduleDynamicTimezone
	result, _, _ := getScheduleTimezone.Call(uintptr(unsafe.Pointer(&zone)))
	if uint32(result) == 0xffffffff {
		return ""
	}
	// A manually disabled seasonal rule is not the standard IANA zone's rules.
	// Do not pretend the named mapping represents that custom configuration.
	if zone.DynamicDaylightTimeDisabled != 0 && zone.StandardDate.Month != 0 {
		return ""
	}
	return windowsScheduleZone(syscall.UTF16ToString(zone.TimeZoneKeyName[:]))
}
