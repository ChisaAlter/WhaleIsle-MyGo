package sidebar

import (
	"time"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/i18n"
	scheduleui "whaleisle/internal/ui/schedule"
)

func scheduleHoverFrequency(row schedstore.Entry, now time.Time) string {
	return scheduleui.FormatCatalogFrequency(row.Raw, now, i18n.Locale())
}
