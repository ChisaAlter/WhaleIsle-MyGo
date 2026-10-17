# PR08: Windows schedule timezone default

Confirmed path: `newTaskDraft` calls `systemScheduleZone`. Go's Windows `time.Local.String()` normally returns `Local`, which is not an IANA ID; the old candidates had only `TZ` and Unix files afterward and therefore usually fell back to UTC.

The new Windows-only reader calls the public, read-only `GetDynamicTimeZoneInformation` API and maps its nonlocalized `TimeZoneKeyName` through Go 1.27.2's CLDR global-territory mapping. Explicit valid `TZ` takes precedence. The normal IANA loader then validates the mapped zone against bundled tzdata. This preserves seasonal rules rather than guessing from the current offset or display language.

Unknown IDs, API failures and custom disabled-DST configurations that contradict a seasonal IANA zone retain the existing explicit UTC fallback. The zone picker remains editable. CLDR's global representative can differ from a user's territory-specific city, so this is a default, not a location detector. No system settings are changed and no task is created or executed.

Sources:
- https://learn.microsoft.com/en-us/windows/win32/api/timezoneapi/nf-timezoneapi-getdynamictimezoneinformation
- https://learn.microsoft.com/en-us/windows/win32/api/timezoneapi/ns-timezoneapi-dynamic_time_zone_information
- Go 1.27.2 `src/time/zoneinfo_windows.go` and generated `src/time/zoneinfo_abbrs_windows.go`; mapping license retained in `docs/licenses/go-time-zone-map-LICENSE`.

Verification: synthetic mapping/catalog/DST fixtures plus Windows compilation. Actual Windows API execution remains unverified in this workspace.
