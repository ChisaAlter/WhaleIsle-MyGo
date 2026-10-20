package interaction

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The recommended option renders the badge instead of the Host's literal
// "(recommended)" words, which the store strips from the display label.
func TestRecommendedOptionBadge(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		questionOptionCard(c, "q0", 0, "Fast", "Short answer", false, false, false, true, nil)
	}, 420, 140)
	if !tt.HasText("推荐") {
		t.Fatal("recommended option is missing its badge")
	}
	if tt.HasText("(recommended)") {
		t.Fatal("the raw suffix should not reach the option card")
	}
}

func TestPlainOptionHasNoBadge(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		questionOptionCard(c, "q0", 0, "Slow", "Take the time", false, false, false, false, nil)
	}, 420, 140)
	if tt.HasText("推荐") {
		t.Fatal("a plain option must not show the recommendation badge")
	}
}
