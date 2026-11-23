//go:build !release

package boot

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestMotionTicksAreShared(t *testing.T) {
	base := motionNow(time.Unix(1000, 0))
	for _, offset := range []time.Duration{0, 3 * time.Millisecond, 20 * time.Millisecond, motionStep - time.Millisecond} {
		now := base.Add(offset)
		next := now.Add(motionWait(now))
		if !motionNow(next).Equal(next) || next.Sub(motionNow(now)) != motionStep {
			t.Fatalf("at +%s the next frame %s is not the next shared tick", offset, next.Sub(base))
		}
		if motionNow(now) != motionNow(base) {
			t.Fatalf("+%s and the tick start read different clocks", offset)
		}
	}
	if fps := float64(time.Second) / float64(motionStep); fps > 30 {
		t.Fatalf("boot motion paced at %.0f fps, above the 30 fps budget", fps)
	}
	if p := loopPhase(base.Add(motionStep/2), time.Second); p != loopPhase(base, time.Second) {
		t.Fatal("loop phases advance only on the shared tick")
	}
}

// TestBootLogDrawerBuildsOnlyVisibleRows: 400 Host lines open at the end of
// the log, and the rows far above the view are not built.
func TestBootLogDrawerBuildsOnlyVisibleRows(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	logs := make([]string, 400)
	for i := range logs {
		logs[i] = fmt.Sprintf("host line %03d", i)
	}
	m := Model{Status: harness.Status{State: harness.Starting}, Logs: logs}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 1200, 800)
	if err := tt.Click("All logs"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("host line 399") {
		t.Fatal("the drawer must follow the newest line")
	}
	if tt.HasText("host line 005") {
		t.Fatal("the drawer built rows far outside its view")
	}
	if !tt.HasText("400 lines") {
		t.Fatal("the drawer title must still count every line")
	}
}

// TestBootPageRequestsNoContinuousFrames: while starting, the page asks only
// for the next shared tick, never a frame per refresh. The tester runs frames
// until the view stops asking (or 20), so a per-frame request shows as 20.
func TestBootPageRequestsNoContinuousFrames(t *testing.T) {
	for _, dark := range []bool{false, true} {
		builds := 0
		m := Model{Status: harness.Status{State: harness.Starting}, Logs: []string{"host line"}, Dark: dark}
		tt := ui.NewTester(func(c *ui.Context) { builds++; View(c, m) }, 1200, 800)
		builds = 0
		tt.Frame()
		if builds > 3 {
			t.Fatalf("dark=%v: one boot frame chained into %d builds; the page is driving continuous redraws", dark, builds)
		}
	}
}
