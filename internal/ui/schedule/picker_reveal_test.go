package schedule

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/i18n"
)

// inkOf returns how much darker than white the area around a box is: a panel
// that has faded in draws its text and borders there, one that has not draws
// nothing.
func inkOf(tt *ui.Tester, box ui.Rect) float64 {
	img := tt.Image()
	total, n := 0.0, 0
	for y := int(box.Y); y < int(box.Y+box.H) && y < img.Bounds().Dy(); y++ {
		for x := int(box.X); x < int(box.X+box.W) && x < img.Bounds().Dx(); x++ {
			r, g, b, _ := img.RGBAAt(x, y).RGBA()
			total += float64(255 - (r+g+b)/3/257)
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return total / float64(n)
}

// The calendar is a selector that opens on the popover token: it fades in over
// 160 ms rather than appearing between two frames.
func TestScheduleCalendarFadesInOverThePopoverToken(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	d := newTaskDraft(schedstore.Entry{Raw: json.RawMessage(`{"kind":"at","scheduledAt":"2026-12-01T01:00:00Z"}`)})
	d.picker.calendarOpen = true
	d.picker.month = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	p := &Panel{recentZones: []string{"UTC"}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(12).Children(func() { p.datePicker(c, d) })
	}, 320, 500)
	base := time.Unix(9700, 0)
	tt.SetTime(base)
	box, ok := tt.Find("31")
	if !ok {
		t.Fatalf("calendar did not open: %q", tt.Texts())
	}
	opened := inkOf(tt, box)
	tt.SetTime(base.Add(80 * time.Millisecond))
	middle := inkOf(tt, box)
	tt.SetTime(base.Add(220 * time.Millisecond))
	settled := inkOf(tt, box)
	if opened != 0 {
		t.Fatalf("calendar was already drawn on the frame it opened: %v", opened)
	}
	if middle <= 0 || middle >= settled {
		t.Fatalf("calendar did not fade in: open=%v middle=%v settled=%v", opened, middle, settled)
	}
	tt.SetTime(base.Add(400 * time.Millisecond))
	if again := inkOf(tt, box); again != settled {
		t.Fatalf("calendar kept moving after it landed: %v then %v", settled, again)
	}
}
