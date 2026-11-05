package sidebar

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSessionMetadataSourceTimeBuckets(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	i18n.SetLocale("en")
	for _, tc := range []struct {
		elapsed float64
		want    string
	}{{-1, "now"}, {59_999, "now"}, {60_000, "1min"}, {3_600_000, "1h"}, {86_400_000, "1d"}, {30 * 86_400_000, "1mo"}, {365 * 86_400_000, "1y"}} {
		if got := sessionRelativeTime(1_000_000, 1_000_000+tc.elapsed); got != tc.want {
			t.Errorf("elapsed %v: %q want %q", tc.elapsed, got, tc.want)
		}
	}
	if got := sessionTrailingLabel("plan-review", 0); got != "Plan review" {
		t.Fatalf("pending interaction did not replace time: %q", got)
	}
	i18n.SetLocale("zh")
	if got := sessionRelativeTime(0, 120_000); got != "2分钟" {
		t.Fatalf("Chinese time=%q", got)
	}
}

func TestSessionMetadataRestHoverMenuAndBlank(t *testing.T) {
	for _, width := range []int{180, 280} {
		item := session.Item{SessionID: "row", Title: "A long pinned conversation title", UpdatedAt: float64(time.Now().Add(-2 * time.Hour).UnixMilli()), Pinned: true}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				rowSession(c, item, 0, Model{OnArchive: func(string) {}}, ui.RGB(30, 30, 30), ui.RGB(100, 100, 100), ui.Transparent)
			})
		}, width, 120)
		row, _ := tt.Find(item.Title)
		stamp, found := tt.Find("Session time row")
		pin, pinned := tt.Find("Session pinned row")
		if !found || !pinned || pin.W != 16 || pin.H != 20 || stamp.X+stamp.W > pin.X || pin.X+pin.W > row.X+row.W-8 {
			t.Fatalf("rest metadata width%d row=%+v stamp=%+v pin=%+v found=%v/%v", width, row, stamp, pin, found, pinned)
		}
		if width == 280 {
			saveSessionMetadataEvidence(t, tt, "sidebar-metadata-rest")
		}
		tt.Move(10, 16)
		if _, ok := tt.Find("Session time row"); ok {
			t.Fatal("hover retained timestamp instead of action cell")
		}
		if _, ok := tt.Find("Session pinned row"); ok {
			t.Fatal("hover retained duplicate pinned marker")
		}
		if width == 280 {
			saveSessionMetadataEvidence(t, tt, "sidebar-metadata-hover")
		}
		if err := tt.Click(sessionActionsLabel("row", "A long pinned conversation title")); err != nil {
			t.Fatal(err)
		}
		tt.Move(10, 100)
		if tt.Menu() == nil {
			t.Fatal("menu did not stay open")
		}
		if _, ok := tt.Find("Session time row"); ok {
			t.Fatal("open menu lost its active action cell after pointer left")
		}
		tt.CloseMenu()
		item.Blank = true
		tt.Frame()
		for _, label := range []string{"Session time row", "Session pinned row", sessionActionsLabel("row", "A long pinned conversation title")} {
			if _, ok := tt.Find(label); ok {
				t.Fatalf("blank provisional row has metadata/action %q", label)
			}
		}
	}
}

func saveSessionMetadataEvidence(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	if dir := os.Getenv("PARITY_GAP_SNAPSHOT_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}
