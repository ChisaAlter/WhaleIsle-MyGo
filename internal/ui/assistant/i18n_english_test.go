package assistant

import (
	"strings"
	"testing"
	"unicode"

	"github.com/egoist/mygo/ui"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func hasHanText(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// TestEnglishLocaleShowsNoChineseCopy guards the assistant surfaces that used to
// be Chinese-only: panel copy, safety text and the pure label helpers.
func TestEnglishLocaleShowsNoChineseCopy(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	check := func(name, text string) {
		t.Helper()
		if hasHanText(text) {
			t.Fatalf("%s still contains Chinese under the English locale: %q", name, text)
		}
	}

	// Pure helpers shared by the panels.
	effects, confirm, cancel, idLabel := botDeleteReviewedLabels("bot")
	check("bot delete effects", effects)
	check("bot delete confirm", confirm)
	check("bot delete cancel", cancel)
	check("bot delete id label", idLabel)
	roomEffects, roomConfirm, roomCancel, roomID := botDeleteReviewedLabels("room")
	check("room delete effects", roomEffects)
	check("room delete confirm", roomConfirm)
	check("room delete cancel", roomCancel)
	check("room delete id label", roomID)

	for _, frequency := range []string{"hourly", "weekdays", "weekly", "monthly", "once", "interval", "daily"} {
		check("frequency "+frequency, routineCreateFrequencyLabel(frequency))
	}
	check("notify enabled", botNotifyValue(true))
	check("notify disabled", botNotifyValue(false))
	check("memory review enabled", botMemoryReviewValue(true))
	check("memory review disabled", botMemoryReviewValue(false))
	check("room runtime open", roomRuntimeOpenLabel())

	// A rendered panel: routine rows carry status, watch and action copy.
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{
			Mode:          assiststore.ModeDshbot,
			HarnessReady:  true,
			DshbotEnabled: true,
			Routines:      RoutineModel{State: emptyRoutineState(), OnRefresh: func() {}},
		}
		routineControlBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
	}, 1100, 3000)
	for _, text := range tt.Texts() {
		if hasHanText(text) {
			t.Fatalf("english routine panel still contains Chinese: %q", strings.TrimSpace(text))
		}
	}
}

func emptyRoutineState() dshbotstore.RoutineSnapshot {
	return dshbotstore.RoutineSnapshot{
		Scope:    dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"},
		Loaded:   true,
		Revision: 1,
		BotName:  "Bot B",
	}
}
