package sessioncost

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	cost "whaleisle/internal/store/sessioncost"
	"whaleisle/internal/ui/i18n"
)

// The composer's statistics strip is one centered row of two clusters. The
// conversation's own performance row must stay off beside it, so this pins the
// single-row contract (see OwnsPerformanceStats).
func TestDockLineIsOneCenteredStatisticsRow(t *testing.T) {
	if !OwnsPerformanceStats {
		t.Fatal("sessioncost no longer owns the performance clusters")
	}
	first, completed := 1000.0, 1600.0
	m := Model{
		State: &State{},
		Snapshot: session.ViewState{
			SelectedID:  "s",
			ClientReady: true,
			TurnUsage:   map[float64]session.TurnUsage{1: {}, 2: {}, 3: {}},
			Messages: []session.Message{{
				Seq:            1,
				TrajectoryTime: &session.TrajectoryTiming{FirstTokenAt: &first, CompletedAt: &completed},
				Raw:            json.RawMessage(`{"type":"assistant/message","data":{"usage":{"outputTokens":300}}}`),
			}},
			CostSources: []cost.Source{{ID: "s", Usage: &cost.Projection{Models: []cost.Route{{
				Provider: "deepseek", Model: "chat",
				Peak:    &cost.Bucket{Hit: 70000, Miss: 7000, Output: 700},
				OffPeak: &cost.Bucket{},
			}}}}},
		},
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m, ui.RGB(0x81, 0x85, 0x8c)) }, 900, 240)
	clusters := fmt.Sprintf("%d %s %d %s", 3, i18n.T(i18n.KeyComposerDockTurns), 1, i18n.T(i18n.KeyComposerDockSteps))
	clusters += " · " + fmt.Sprintf("%.0f %s", 500.0, i18n.T(i18n.KeyComposerDockSpeedUnit))
	tokens := fmt.Sprintf("77.7K %s · %s", i18n.T(i18n.KeyComposerDockTokenUnit),
		fmt.Sprintf(i18n.T(i18n.KeyComposerDockCacheHit), "90.9%"))
	counts, ok := tt.Find(clusters)
	if !ok {
		t.Fatalf("dock lost the counts cluster %q: %v", clusters, tt.Texts())
	}
	usage, ok := tt.Find(tokens)
	if !ok {
		t.Fatalf("dock lost the token cluster %q: %v", tokens, tt.Texts())
	}
	if math.Abs(float64((counts.Y+counts.H/2)-(usage.Y+usage.H/2))) > 1 {
		t.Fatalf("clusters are not on one row: %v %v", counts, usage)
	}
	left := math.Min(float64(counts.X), float64(usage.X))
	right := math.Max(float64(counts.X+counts.W), float64(usage.X+usage.W))
	if center := (left + right) / 2; math.Abs(center-450) > 12 {
		t.Fatalf("statistics row is not centered: center=%f", center)
	}
	speed := i18n.T(i18n.KeyComposerDockSpeedUnit)
	speedLines := 0
	for _, text := range tt.Texts() {
		if strings.Contains(text, speed) {
			speedLines++
		}
	}
	if speedLines != 1 {
		t.Fatalf("dock painted %d speed rows, want 1: %v", speedLines, tt.Texts())
	}
}
