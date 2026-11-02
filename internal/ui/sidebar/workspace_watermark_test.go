package sidebar

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func watermarkFixture(count int) (Model, session.WorkspaceGroup) {
	var items []session.Item
	var ids []string
	for i := 1; i <= count; i++ {
		id := fmt.Sprintf("row-%02d", i)
		ids = append(ids, id)
		items = append(items, session.Item{SessionID: id, Title: id, Cwd: "/repo", UpdatedAt: float64(count - i)})
	}
	workspace := session.WorkspaceRow{WorkspaceID: "one", Path: "/repo", SessionIDs: ids}
	m := Model{Snapshot: session.ViewState{WorkspaceReady: true, Items: items, Workspaces: []session.WorkspaceRow{workspace}, WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 7, GroupBy: "workspace", OrderBy: "manual", Collapsed: map[string]bool{"one": false}}}, workspaceWatermarks: &workspaceWatermarkState{HostGeneration: 7, Marks: map[string]int{}}}
	return m, session.WorkspaceGroup{Key: "one", Kind: "workspace", Workspace: workspace, Items: items}
}

func TestWorkspaceWatermarkBlankFiveAndLocalMoreLess(t *testing.T) {
	m, group := watermarkFixture(12)
	blank := session.Item{SessionID: "blank", Blank: true}
	group.Items = append(group.Items, blank)
	m.Snapshot.Items = append(m.Snapshot.Items, blank)
	visible := visibleWorkspaceSessionRows(m, group)
	if len(visible.Rows) != 6 || visible.Hidden != 7 || visible.Rows[0].Item.SessionID != "blank" || visible.Watermark != 5 {
		t.Fatalf("blank must not consume five-row budget: %+v", visible)
	}
	retainWorkspaceWatermark(m, group.Key, visible.Watermark+workspaceWatermarkBase)
	visible = visibleWorkspaceSessionRows(m, group)
	if len(visible.Rows) != 11 || visible.Hidden != 2 {
		t.Fatal("Show more did not add five nonblank rows")
	}
	retainWorkspaceWatermark(m, group.Key, visible.Watermark+workspaceWatermarkBase)
	if got := visibleWorkspaceSessionRows(m, group); got.Hidden != 0 || len(got.Rows) != 13 {
		t.Fatal("last page was not fully revealed")
	}
	resetWorkspaceWatermark(m, group.Key, nil)
	if got := visibleWorkspaceSessionRows(m, group); got.Watermark != 5 || got.Hidden != 7 {
		t.Fatal("Show less failed to return to the five-row baseline")
	}
	retainWorkspaceWatermark(m, group.Key, 30)
	m.Snapshot.WorkspaceLayout.HostGeneration++
	if got := visibleWorkspaceSessionRows(m, group); got.Watermark != 5 {
		t.Fatal("another Host inherited stored watermark")
	}
}

func TestWorkspaceWatermarkAttentionAutoFiftyAndUnboundedSearchReveal(t *testing.T) {
	m, group := watermarkFixture(65)
	group.Items[11].CompletionUnread = true
	group.Items[49].Running = true
	group.Items[54].Running = true
	m.Snapshot.Items = append([]session.Item(nil), group.Items...)
	m.Snapshot.SelectedID = "row-61"
	m.PendingStatuses = map[string]string{"row-30": "question"}
	if got := visibleWorkspaceSessionRows(m, group); got.Baseline != 50 || len(got.Rows) != 50 || got.Hidden != 15 {
		t.Fatalf("automatic attention must stop at fifty, including current: %+v", got)
	}
	m.searchState = &sessionSearchState{Reveal: "row-63"}
	if got := visibleWorkspaceSessionRows(m, group); got.Baseline != 63 || len(got.Rows) != 63 || got.Hidden != 2 {
		t.Fatalf("search reveal should exceed automatic maximum: %+v", got)
	}
	m.searchState.Reveal = "missing"
	group.Items[49].Running = false
	group.Items[54].Running = false
	m.Snapshot.Items = append([]session.Item(nil), group.Items...)
	m.Snapshot.Items = append(m.Snapshot.Items, session.Item{SessionID: "running-fork", ParentID: "row-48", Running: true})
	if got := visibleWorkspaceSessionRows(m, group); got.Baseline != 30 {
		t.Fatal("ordinary running fork incorrectly raised its parent's watermark")
	}
	m.Snapshot.Items = append(m.Snapshot.Items, session.Item{SessionID: "running-child", Origin: "subagent", ParentID: "row-42", Running: true})
	if got := visibleWorkspaceSessionRows(m, group); got.Baseline != 42 {
		t.Fatal("running child did not raise its visible parent's watermark")
	}
}

func TestWorkspaceWatermarkFoldResetsDescendantsAndFlatIsUnbounded(t *testing.T) {
	m, group := watermarkFixture(12)
	groups := []session.WorkspaceGroup{group, {Key: "child", ParentID: "one"}, {Key: "grandchild", ParentID: "child"}, {Key: "other"}}
	for _, key := range []string{"one", "child", "grandchild", "other"} {
		retainWorkspaceWatermark(m, key, 20)
	}
	resetWorkspaceWatermark(m, "one", groups)
	if len(m.workspaceWatermarks.Marks) != 1 || m.workspaceWatermarks.Marks["other"] != 20 {
		t.Fatal("fold reset lost another group or retained hidden descendant watermark")
	}
	group.Kind = "flat"
	if got := visibleWorkspaceSessionRows(m, group); len(got.Rows) != 12 || got.Hidden != 0 {
		t.Fatal("flat source list was truncated")
	}
}

func TestWorkspaceWatermarkArchivedPinsCannotPromoteRows(t *testing.T) {
	m, group := watermarkFixture(8)
	m.Snapshot.PinnedSessionIDs = []string{"row-08"}
	m.Snapshot.ArchivedSessionIDs = []string{"row-08"}
	if got := visibleWorkspaceSessionRows(m, group); got.Rows[0].Item.SessionID != "row-01" || len(got.Rows) != 5 {
		t.Fatal("an archived pin escaped the ordinary source order")
	}
}

func TestWorkspaceWatermarkRenderedControlsUseActualVisibleRows(t *testing.T) {
	m, _ := watermarkFixture(12)
	m.OnToggleWorkspaceGroup = func(id string) {
		m.Snapshot.WorkspaceLayout.Collapsed[id] = !session.WorkspaceGroupCollapsed(m.Snapshot, id)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		workspaceRowsView(c, m, m.Snapshot.Items, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), ui.RGB(0, 0, 0))
	}, 300, 1500)
	if !tt.HasText(workspaceText("展开其余 7 个会话", "Show 7 more sessions")) {
		t.Fatal("overflow label did not expose the remaining count")
	}
	if !tt.HasText("row-05") || tt.HasText("row-06") {
		t.Fatal("render did not stop at the five-row baseline")
	}
	if err := tt.Click(workspaceText("显示更多", "Show more") + " one"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(workspaceText("展开其余 2 个会话", "Show 2 more sessions")) {
		t.Fatal("overflow label retained the previous hidden count")
	}
	if !tt.HasText("row-10") || tt.HasText("row-11") {
		t.Fatal("render did not reveal exactly five further rows")
	}
	if err := tt.Click(workspaceText("显示更多", "Show more") + " one"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("row-12") {
		t.Fatal("last page not rendered")
	}
	if err := tt.Click(workspaceText("收起会话", "Show less") + " one"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("row-06") {
		t.Fatal("Show less retained an overflow row")
	}
}
