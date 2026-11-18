package sidebar

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSidebarFlatModeVirtualizesAndScrolls(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() {
		i18n.SetLocale(i18n.DefaultLocale)
		sidebarVirtualRowHook = nil
	})

	const total = 120
	items := make([]session.Item, total)
	ids := make([]string, total)
	for i := range items {
		id := fmt.Sprintf("flat-%03d", i)
		ids[i] = id
		items[i] = session.Item{SessionID: id, Title: fmt.Sprintf("Flat Row %03d", i)}
	}
	m := Model{
		ThemeID:         "deepseek.light",
		ViewportW:       320,
		ViewportH:       360,
		ConnectionReady: true,
		Snapshot: session.ViewState{
			WorkspaceReady: true,
			Workspaces:     []session.WorkspaceRow{{WorkspaceID: "ws", Title: "Workspace", SessionIDs: ids}},
			Items:          items,
			WorkspaceLayout: session.WorkspaceLayoutState{
				GroupBy: "flat",
			},
		},
	}

	built := map[int]string{}
	sidebarVirtualRowHook = func(surface string, index int, id string) {
		if surface == "flat" {
			built[index] = id
		}
	}

	tt := ui.NewTester(func(c *ui.Context) {
		clear(built)
		View(c, m)
	}, 320, 360)
	clear(built)
	tt.Frame()

	if len(built) == 0 || len(built) >= 22 {
		t.Fatalf("flat viewport built %d rows on initial frame, want bounded subset (<22)", len(built))
	}
	if !tt.HasText("Flat Row 000") || tt.HasText("Flat Row 045") {
		t.Fatalf("flat viewport initial visibility wrong: has0=%v has45=%v built=%d", tt.HasText("Flat Row 000"), tt.HasText("Flat Row 045"), len(built))
	}

	firstRect, ok := tt.Find("Flat Row 000")
	if !ok {
		t.Fatal("missing first flat row before wheel scroll")
	}
	for range 8 {
		tt.Scroll(firstRect.X+10, firstRect.Y+10, 0, 240)
	}
	clear(built)
	tt.Frame()

	if len(built) == 0 || len(built) >= 24 {
		t.Fatalf("flat viewport built %d rows after scroll, want bounded subset (<24)", len(built))
	}
	if tt.HasText("Flat Row 000") {
		t.Fatal("top flat row stayed mounted after scrolling down")
	}
	scrolledTarget := false
	for idx := range built {
		if idx >= 15 {
			scrolledTarget = true
			break
		}
	}
	if !scrolledTarget {
		t.Fatalf("wheel scroll did not advance flat list rows: built=%v", built)
	}
}

func TestSidebarArchivedSectionVirtualizesAndScrolls(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() {
		i18n.SetLocale(i18n.DefaultLocale)
		sidebarVirtualRowHook = nil
	})

	const total = 120
	items := make([]session.Item, total+1)
	archivedIDs := make([]string, total)
	items[0] = session.Item{SessionID: "active-1", Title: "Active Session"}
	for i := 0; i < total; i++ {
		id := fmt.Sprintf("arch-%03d", i)
		archivedIDs[i] = id
		items[i+1] = session.Item{SessionID: id, Title: fmt.Sprintf("Archived Row %03d", i)}
	}
	m := Model{
		ThemeID:         "deepseek.light",
		ViewportW:       320,
		ViewportH:       360,
		ConnectionReady: true,
		Snapshot: session.ViewState{
			WorkspaceReady:     true,
			Workspaces:         []session.WorkspaceRow{{WorkspaceID: "ws", Title: "Workspace", SessionIDs: append([]string{"active-1"}, archivedIDs...)}},
			ArchivedSessionIDs: archivedIDs,
			Items:              items,
		},
	}

	built := map[int]string{}
	sidebarVirtualRowHook = func(surface string, index int, id string) {
		if surface == "archive" {
			built[index] = id
		}
	}

	tt := ui.NewTester(func(c *ui.Context) {
		clear(built)
		View(c, m)
	}, 320, 360)

	if err := tt.Click(i18n.T(i18n.KeySidebarArchivedSection)); err != nil {
		t.Fatalf("expand archived section failed: %v", err)
	}
	clear(built)
	tt.Frame()

	if len(built) == 0 || len(built) >= 20 {
		t.Fatalf("expanded archived list built %d rows, want bounded subset (<20)", len(built))
	}
	if !tt.HasText("Archived Row 000") || tt.HasText("Archived Row 025") {
		t.Fatalf("expanded archived visibility wrong: has0=%v has25=%v built=%d", tt.HasText("Archived Row 000"), tt.HasText("Archived Row 025"), len(built))
	}

	firstArch, ok := tt.Find("Archived Row 000")
	if !ok {
		t.Fatal("missing first archived row before wheel scroll")
	}
	for range 6 {
		tt.Scroll(firstArch.X+10, firstArch.Y+10, 0, 240)
	}
	clear(built)
	tt.Frame()

	if len(built) == 0 || len(built) >= 22 {
		t.Fatalf("archived list built %d rows after scroll, want bounded subset (<22)", len(built))
	}
	if tt.HasText("Archived Row 000") {
		t.Fatal("top archived row stayed mounted after scrolling down")
	}
}

func TestSidebarSearchResultsVirtualizesAndScrolls(t *testing.T) {
	i18n.SetLocale("en")
	t.Cleanup(func() {
		i18n.SetLocale(i18n.DefaultLocale)
		sidebarVirtualRowHook = nil
	})

	const total = 80
	items := make([]session.Item, total)
	ids := make([]string, total)
	for i := range items {
		id := fmt.Sprintf("hit-%03d", i)
		ids[i] = id
		items[i] = session.Item{SessionID: id, Title: fmt.Sprintf("needle result %03d", i)}
	}
	query := "needle"
	m := Model{
		ThemeID:         "deepseek.light",
		ViewportW:       320,
		ViewportH:       320,
		ConnectionReady: true,
		SearchQuery:     &query,
		Snapshot: session.ViewState{
			WorkspaceReady: true,
			Workspaces:     []session.WorkspaceRow{{WorkspaceID: "ws", Title: "Workspace", SessionIDs: ids}},
			Items:          items,
		},
	}

	built := map[int]string{}
	sidebarVirtualRowHook = func(surface string, index int, id string) {
		if surface == "search" {
			built[index] = id
		}
	}

	tt := ui.NewTester(func(c *ui.Context) {
		clear(built)
		View(c, m)
	}, 320, 320)
	clear(built)
	tt.Frame()

	if len(built) == 0 || len(built) >= 16 {
		t.Fatalf("search results viewport built %d rows, want bounded subset (<16)", len(built))
	}
	if !tt.HasText("needle result 000") || tt.HasText("needle result 040") {
		t.Fatalf("search viewport initial visibility wrong: has0=%v has40=%v built=%d", tt.HasText("needle result 000"), tt.HasText("needle result 040"), len(built))
	}

	firstHit, ok := tt.Find("needle result 000")
	if !ok {
		t.Fatal("missing first search hit before wheel scroll")
	}
	for range 8 {
		tt.Scroll(firstHit.X+10, firstHit.Y+10, 0, 240)
	}
	clear(built)
	tt.Frame()

	if len(built) == 0 || len(built) >= 18 {
		t.Fatalf("search results built %d rows after scroll, want bounded subset (<18)", len(built))
	}
	if tt.HasText("needle result 000") {
		t.Fatal("top search result stayed mounted after scrolling down")
	}
	scrolledTarget := false
	for idx := range built {
		if idx >= 10 {
			scrolledTarget = true
			break
		}
	}
	if !scrolledTarget {
		t.Fatalf("wheel scroll did not advance search results: built=%v", built)
	}
}
