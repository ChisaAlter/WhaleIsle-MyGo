package sessioncost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	cost "whaleisle/internal/store/sessioncost"
	"whaleisle/internal/ui/i18n"
)

func TestHistoryDerivedDockRespectsTimingMutationsAndHiddenStats(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	first, end := 1000.0, 1600.0
	m := Model{Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "host-a/follow", TurnUsage: map[float64]session.TurnUsage{1: {}}, Messages: []session.Message{{Seq: 1, Raw: json.RawMessage(`{"type":"assistant/message","data":{"usage":{"outputTokens":300}}}`), TrajectoryTime: &session.TrajectoryTiming{FirstTokenAt: &first, CompletedAt: &end}}}}}
	stats := true
	tt := ui.NewTester(func(c *ui.Context) { dockLine(c, m, cost.Result{}, stats, c.Theme().TextMuted) }, 700, 160)
	assertSpeed := func(value float64) {
		t.Helper()
		if !tt.HasText(fmt.Sprintf("%.0f %s", value, i18n.T(i18n.KeyComposerDockSpeedUnit))) {
			t.Fatal("current dock speed missing", tt.Texts())
		}
	}
	assertSpeed(500)
	for range 3 {
		tt.Frame()
		assertSpeed(500)
	}
	raw := m.Snapshot.Messages[0].Raw
	at := bytes.Index(raw, []byte("300"))
	copy(raw[at:at+3], "600")
	tt.Frame()
	assertSpeed(1000)
	*m.Snapshot.Messages[0].TrajectoryTime.CompletedAt = 2200
	tt.Frame()
	assertSpeed(500)
	stats = false
	tt.Frame()
	if len(tt.Texts()) != 0 {
		t.Fatal("disabled stats remained visible", tt.Texts())
	}
}

// Disabled stats must not even traverse source rows. This direct dock microbench
// excludes rasterization and measures the early return for a large loaded window.
func BenchmarkHistoryDerivedDockStatsDisabled(b *testing.B) {
	m := Model{Snapshot: session.ViewState{SelectedID: "s", Messages: make([]session.Message, 2000)}}
	for i := range m.Snapshot.Messages {
		m.Snapshot.Messages[i].Raw = json.RawMessage(`{"type":"assistant/message","data":{"usage":{"outputTokens":300}}}`)
	}
	b.ReportAllocs()
	for range b.N {
		dockLine(nil, m, cost.Result{}, false, ui.Color{})
	}
}

func TestHistoryDerivedDockHiddenDoesNotRequireUIContext(t *testing.T) {
	m := Model{Snapshot: session.ViewState{Messages: []session.Message{{Raw: json.RawMessage(`{"type":"assistant/message"}`)}}}}
	dockLine(nil, m, cost.Result{}, false, ui.Color{})
}
