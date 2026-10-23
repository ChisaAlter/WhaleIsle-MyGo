package chat

import (
	"fmt"
	"math"
	"time"

	"whaleisle/internal/ui/i18n"
)

// formatMessageClock uses the event's local calendar day, as transcript chrome
// does in the reference. The trajectory's inspection clock is separate.
func formatMessageClock(ms float64, now time.Time) string {
	if math.IsNaN(ms) || math.IsInf(ms, 0) || math.Abs(ms) > 8.64e15 {
		return ""
	}
	d := time.UnixMilli(int64(ms)).In(now.Location())
	clock := d.Format("15:04")
	if d.Year() == now.Year() && d.YearDay() == now.YearDay() {
		return clock
	}
	if d.Year() == now.Year() {
		return fmt.Sprintf(i18n.T(i18n.KeyChatClockMD), int(d.Month()), d.Day()) + " " + clock
	}
	return fmt.Sprintf(i18n.T(i18n.KeyChatClockYMD), d.Year(), int(d.Month()), d.Day()) + " " + clock
}
