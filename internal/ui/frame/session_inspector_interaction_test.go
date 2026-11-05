package frame

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestInspectorFilterConfirmationAlwaysResetsTable(t *testing.T) {
	for _, query := range []string{" *TOOL* ", "TOOL", "***", ""} {
		p := SessionInspectorPaneState{
			Filter: inspectorFilterQuery(query), selected: "old", breadcrumb: []string{"old", "data"},
			collapsed: map[string]inspectorDisclosure{"parent": {collapsed: true}}, detailsPercent: 80, detailsDragging: true,
			list: ui.ListState{FollowEnd: true}, tableX: ui.ScrollState{X: 30}, tableY: ui.ScrollState{Y: 600},
		}
		p.applyInspectorFilter(query)
		if p.Filter != inspectorFilterQuery(query) || p.draft != p.Filter || len(p.collapsed) != 0 ||
			p.selected != "" || len(p.breadcrumb) != 0 || p.detailsPercent != 0 || p.detailsDragging ||
			p.list.FollowEnd || p.tableY.Y != 0 || p.tableX.X != 30 {
			t.Fatalf("confirmation retained the previous view for %q: %+v", query, p)
		}
	}
}

func TestInspectorFilteredDisclosuresAndDraftCancellation(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	rows := []session.InspectorRecord{
		{Key: "event:1", Type: "turn/start", Seq: 1},
		{Key: "event:2", Parent: "event:1", Type: "step/start", Seq: 2},
		{Key: "event:3", Parent: "event:2", Type: "future/custom", Seq: 3},
	}
	p := SessionInspectorPaneState{}
	p.applyInspectorFilter("custom")
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 600)
	if err := tt.Click("Collapse children 2"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("future/custom") || !tt.HasText("1 / 3") {
		t.Fatal("filtered branch did not fold while retaining its match count")
	}
	if err := tt.Click("Inspector type filter"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Clear"); err != nil {
		t.Fatal(err)
	}
	if p.draft != "" || p.Filter != "custom" {
		t.Fatal("Clear applied the draft")
	}
	if err := tt.Click("Cancel"); err != nil {
		t.Fatal(err)
	}
	if p.Filter != "custom" || !p.collapsed["event:2"].collapsed || tt.HasText("future/custom") {
		t.Fatal("cancelling a cleared draft changed the confirmed hierarchy")
	}
	p.applyInspectorFilter(" *TURN/START* ")
	tt.Frame()
	if _, ok := tt.Find("Collapse children 1"); ok {
		t.Fatal("filtered node with no retained children still has a disclosure")
	}
	if !tt.HasText("turn/start") || !tt.HasText("1 / 3") {
		t.Fatal("matching parent disappeared with its excluded descendants")
	}
}

func TestInspectorPickPreservesFilterAndOtherFolds(t *testing.T) {
	rows := []session.InspectorRecord{
		{Key: "other", Type: "turn"}, {Key: "other-leaf", Parent: "other", Type: "match"},
		{Key: "turn", Type: "turn"}, {Key: "step", Parent: "turn", Type: "step"},
		{Key: "target", Parent: "step", Type: "match", Seq: 9}, {Key: "detail", Parent: "target", Type: "match"},
	}
	p := SessionInspectorPaneState{Filter: "match", draft: "match", collapsed: map[string]inspectorDisclosure{"other": {state: "open", collapsed: true}, "turn": {state: "open", collapsed: true}, "step": {state: "open", collapsed: true}, "target": {state: "open", collapsed: true}},
		navigation: chat.InspectorNavigation{Picking: true, Picked: &chat.InspectorTarget{Seq: 9}, Target: &chat.InspectorTarget{ID: "old"}, Notice: "old miss"}}
	p.consumeInspectorPick(rows)
	if p.Filter != "match" || p.draft != "match" || p.collapsed["turn"].collapsed || p.collapsed["step"].collapsed || !p.collapsed["other"].collapsed || !p.collapsed["target"].collapsed {
		t.Fatalf("pick reset the filter or unrelated/target disclosure: %+v", p)
	}
	if p.selected != "target" || len(p.breadcrumb) != 1 || p.navigation.Target != nil || p.navigation.Picked != nil || p.navigation.Picking || p.navigation.Notice != "" {
		t.Fatalf("pick left stale navigation: %+v", p.navigation)
	}
	// Contextual ancestors are included by the filter, even though their type does not match.
	if !p.revealInspectorRow("turn", rows) || p.Filter != "match" {
		t.Fatal("revealing context unnecessarily cleared the filter")
	}
	p.Filter, p.draft = "absent", "absent"
	if !p.revealInspectorRow("target", rows) || p.Filter != "" || p.draft != "" || !p.collapsed["other"].collapsed || !p.collapsed["target"].collapsed {
		t.Fatal("excluded target did not clear only the filter and its ancestor folds")
	}
}

func TestInspectorPickCentersVisibleIndexOnceAndCanRepeat(t *testing.T) {
	var rows []session.InspectorRecord
	for i := 0; i < 80; i++ {
		rows = append(rows, session.InspectorRecord{Key: fmt.Sprintf("excluded:%d", i), Type: "excluded"})
	}
	for i := 0; i < 60; i++ {
		rows = append(rows, session.InspectorRecord{Key: fmt.Sprintf("match:%d", i), Type: "match", Seq: float64(i + 100)})
	}
	p := SessionInspectorPaneState{Filter: "match"}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 600)
	pick := func() {
		t.Helper()
		p.navigation.Picked = &chat.InspectorTarget{Seq: 130}
		tt.Frame()
		header, _ := tt.Find("Inspector table header")
		divider, _ := tt.Find("Inspector details divider")
		row, ok := tt.Find("Inspector record match:30")
		center := (header.Y + header.H + divider.Y) / 2
		if !ok || row.Y+row.H/2 < center-1 || row.Y+row.H/2 > center+1 {
			t.Fatalf("pick did not center in the filtered hierarchy: row=%+v center=%v scroll=%+v", row, center, p.tableY)
		}
	}
	pick()
	header, _ := tt.Find("Inspector table header")
	tt.Scroll(500, header.Y+header.H+60, 0, 120)
	tt.Frame()
	moved := p.tableY.Y
	tt.Frame()
	if p.tableY.Y != moved {
		t.Fatal("render reapplied a consumed pick")
	}
	pick()
	p.closeInspectorDetails()
	tt.Frame()
	if p.selected != "" {
		t.Fatal("closing details replayed an old pick")
	}
}

func TestInspectorSelectionDetailsAndBreadcrumbLifetime(t *testing.T) {
	rows := []session.InspectorRecord{{Key: "chat:root", Type: "root", Seq: 1}, {Key: "chat:next", Type: "next", Seq: 2}}
	p := SessionInspectorPaneState{Mode: "chat-node", list: ui.ListState{FollowEnd: true}, detailsPercent: 65}
	p.selectInspectorRow(rows[0])
	if !p.list.FollowEnd || p.detailsPercent != 65 {
		t.Fatal("ordinary selection changed follow policy or resized details")
	}
	p.followInspectorReference("chat:next", rows)
	p.backInspectorReference(0, rows)
	if p.selected != "chat:root" || len(p.breadcrumb) != 1 || p.navigation.Target.ID != "root" || p.list.FollowEnd || p.detailsPercent != 65 {
		t.Fatal("breadcrumb back did not restore row, chat target and existing details")
	}
	p.closeInspectorDetails()
	if p.navigation.Target == nil || p.inspectorDetailsPercent() != 40 || p.selected != "" {
		t.Fatal("Close cancelled chat reveal or retained the resized detail lifetime")
	}
	p.selectInspectorRow(rows[1])
	if p.inspectorDetailsPercent() != 40 {
		t.Fatal("new details did not start at 40 percent")
	}
	p.toggleInspectorPicking()
	if !p.navigation.Picking || p.navigation.Target != nil {
		t.Fatal("starting picker retained a previous chat reveal")
	}
}

func TestInspectorRemovedSelectionKeepsDetails(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	p := SessionInspectorPaneState{selected: "missing", breadcrumb: []string{"missing"}, detailsPercent: 65}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 600)
	if !tt.HasText("This record was removed or settled. Select a current record.") || p.inspectorDetailsPercent() != 65 {
		t.Fatal("removed record silently closed details")
	}
	closeBounds, _ := tt.Find("Close")
	dividerBounds, _ := tt.Find("Inspector details divider")
	if err := tt.Click("Close"); err != nil || p.selected != "" {
		t.Fatalf("removed details cannot close: %v selected=%q close=%+v divider=%+v", err, p.selected, closeBounds, dividerBounds)
	}
}

func TestInspectorNavigationUsesActualResourceVisibility(t *testing.T) {
	s := session.New()
	state := func() *SessionInspectorPaneState {
		return &SessionInspectorPaneState{binding: s.InspectorBinding(), navigation: chat.InspectorNavigation{Picking: true, Target: &chat.InspectorTarget{ID: "stale"}, Picked: &chat.InspectorTarget{ID: "stale"}}, pendingPick: &chat.InspectorTarget{ID: "stale"}, pendingPickBinding: s.InspectorBinding()}
	}
	hidden := &WorkspacePane{ID: "hidden", Kind: "session-inspector-log", Inspector: state()}
	floating := &WorkspacePane{ID: "float", Kind: "session-inspector-log", Inspector: state(), Float: &WorkspaceFloat{}}
	first := &WorkspacePane{ID: "files", Kind: "files", DockTabs: []*WorkspacePane{hidden, floating}}
	second := &WorkspacePane{ID: "sibling", Kind: "session-inspector-log", Inspector: state()}
	w := &WorkspacePanes{Root: &WorkspacePane{First: first, Second: second}, Active: "sibling", Fullscreen: "files"}
	m := Model{Store: s, Sizes: &LayoutSizes{Workspace: w}, RightPaneOpen: true}
	if got := sessionInspectorChatNavigation(m); got != &second.Inspector.navigation || !got.Picking {
		t.Fatal("fullscreen cancelled a visible sibling's picker")
	}
	if hidden.Inspector.pendingPick != nil || hidden.Inspector.navigation.Picked != nil || hidden.Inspector.navigation.Picking || floating.Inspector.navigation.Picking {
		t.Fatal("hidden tab or competing picker kept a queued gesture")
	}
	m.RightPaneOpen = false
	floating.Inspector.navigation.Picking = true
	if sessionInspectorChatNavigation(m) != &floating.Inspector.navigation {
		t.Fatal("visible float in a dock tab did not attach with dock closed")
	}
	floating.Float.Minimized = true
	if sessionInspectorChatNavigation(m) != nil || floating.Inspector.navigation.Picking {
		t.Fatal("minimized float retained its picker")
	}
}

func TestInspectorOwnerHandoffDisposesConsumedChatReveal(t *testing.T) {
	s := session.New()
	s.Select("current")
	makePane := func(id string) *WorkspacePane {
		return &WorkspacePane{ID: id, Kind: "session-inspector-log", Inspector: &SessionInspectorPaneState{binding: s.InspectorBinding(), navigation: chat.InspectorNavigation{SessionID: "current"}}}
	}
	a, b := makePane("a"), makePane("b")
	w := &WorkspacePanes{Root: &WorkspacePane{ID: "split", First: a, Second: b}, Active: "a"}
	m := Model{Store: s, Sizes: &LayoutSizes{Workspace: w}, RightPaneOpen: true}
	a.Inspector.navigation.Highlight = "consumed-pending-target"
	w.Active = "b"
	if sessionInspectorChatNavigation(m) != &b.Inspector.navigation || a.Inspector.navigation.Highlight != "" {
		t.Fatal("previous Chat owner retained consumed reveal")
	}
	w.Active = "a"
	if sessionInspectorChatNavigation(m) != &a.Inspector.navigation || a.Inspector.navigation.Highlight != "" {
		t.Fatal("owner handoff revived previous reveal")
	}
}

func TestInspectorVisibleOccurrencesShareCurrentChatProjection(t *testing.T) {
	s := session.New()
	s.Select("current")
	makePane := func(id string) *WorkspacePane {
		return &WorkspacePane{ID: id, Kind: "session-inspector-log", Inspector: &SessionInspectorPaneState{binding: s.InspectorBinding(), navigation: chat.InspectorNavigation{SessionID: "current", Nodes: []session.Message{{ID: "old"}}}}}
	}
	first, second, hidden := makePane("first"), makePane("second"), makePane("hidden")
	first.DockTabs = []*WorkspacePane{hidden}
	second.Float = &WorkspaceFloat{}
	w := &WorkspacePanes{Root: &WorkspacePane{ID: "split", First: first, Second: second}}
	m := Model{Store: s, Sizes: &LayoutSizes{Workspace: w}, RightPaneOpen: true}
	source := &first.Inspector.navigation
	source.Nodes = []session.Message{{ID: "current-node"}, {ID: "current-group"}}
	source.Groups = map[string][]string{"current-group": {"current-node"}}
	syncInspectorChatProjection(m, source)
	if len(second.Inspector.navigation.Nodes) != 2 || second.Inspector.navigation.Groups["current-group"][0] != "current-node" {
		t.Fatal("inactive visible inspector retained a stale Chat projection")
	}
	if hidden.Inspector.navigation.Nodes[0].ID != "old" {
		t.Fatal("hidden inspector participated in live projection")
	}
	source.Nodes = nil
	source.Groups = nil
	syncInspectorChatProjection(m, source)
	if len(second.Inspector.navigation.Nodes) != 0 || len(second.Inspector.navigation.Groups) != 0 {
		t.Fatal("cleared Chat retained the previous projection")
	}
}

func TestInspectorInactivePageCancelsGesturesAndCachedProjection(t *testing.T) {
	p := &SessionInspectorPaneState{navigation: chat.InspectorNavigation{Picking: true, Picked: &chat.InspectorTarget{ID: "stale"}, Target: &chat.InspectorTarget{ID: "stale"}, Highlight: "stale", Nodes: []session.Message{{ID: "stale"}}}, pendingPick: &chat.InspectorTarget{ID: "stale"}}
	p.objects = map[string]inspectorObject{"chat:stale": {Value: &p.navigation.Nodes[0]}}
	w := &WorkspacePanes{Root: &WorkspacePane{ID: "inspector", Kind: "session-inspector-log", Inspector: p}}
	w.BeginContentFrame("owner", "root")
	if len(p.navigation.Nodes) != 0 {
		t.Fatal("absent Chat kept an old projection")
	}
	w.EndContentFrame()
	if p.navigation.Picking || p.navigation.Picked != nil || p.navigation.Target != nil || p.pendingPick != nil || p.navigation.Highlight != "" || len(p.objects) != 0 {
		t.Fatal("application page hiding Chat/inspector retained old gestures")
	}
	w.BeginContentFrame("owner", "root")
	p.rendered, p.pickQueued = true, true
	p.pendingPick = &chat.InspectorTarget{ID: "just-routed"}
	w.EndContentFrame()
	if p.pendingPick == nil {
		t.Fatal("new route was cancelled before its first render")
	}
	w.BeginContentFrame("owner", "root")
	p.rendered = true
	p.navigation.Picking = true
	w.EndContentFrame()
	if p.pendingPick != nil || p.navigation.Picking {
		t.Fatal("inspector with no displayed Chat retained the old request")
	}
}

func TestInspectToolCallQueuesOnlyRoutedOccurrenceAndBinding(t *testing.T) {
	s := session.New()
	s.Select("current")
	other := &WorkspacePane{ID: "other", Kind: "session-inspector-log", Inspector: &SessionInspectorPaneState{Mode: "session-log", binding: s.InspectorBinding(), navigation: chat.InspectorNavigation{Picking: true, Target: &chat.InspectorTarget{ID: "old"}}}}
	active := &WorkspacePane{ID: "active", Kind: "files"}
	w := &WorkspacePanes{Root: &WorkspacePane{ID: "split", First: other, Second: active}, Active: "active", route: "files"}
	tab := "files"
	open := false
	m := Model{Store: s, Sizes: &LayoutSizes{Workspace: w}, RightPaneTab: &tab, OnToggleTools: func() { open = !open }}
	m.inspectToolCall("opaque-child", "tool-owner", 9)
	if !open {
		t.Fatal("inspect action did not persist opening the resource dock")
	}
	if other.Inspector.navigation.Picking || other.Inspector.navigation.Target != nil {
		t.Fatal("explicit tool inspection did not replace the previous picker")
	}
	routed := w.Root.find(w.Active).Inspector
	if routed == nil || routed.pendingPick == nil || routed.pendingPick.CallID != "opaque-child" || routed.pendingPickBinding != s.InspectorBinding() || other.Inspector.pendingPick != nil || other.Inspector.Mode != "session-log" {
		t.Fatalf("inspect request was not limited to routed owner: active=%s routed=%+v other=%+v expected=%s", w.Active, routed, other.Inspector, s.InspectorBinding())
	}
	m.RightPaneOpen = true
	w.BeginContentFrame("owner", "root")
	current := sessionInspectorChatNavigation(m)
	syncInspectorChatProjection(m, current)
	if routed.pendingPick == nil || !routed.chatProjected {
		t.Fatal("Chat ownership handoff discarded a new unbound pane's request")
	}
	routed.bindInspectorSource(s.InspectorBinding(), "current")
	routed.rendered = true
	w.EndContentFrame()
	if routed.pendingPick == nil {
		t.Fatal("intentional first-render request lost its owner")
	}
	s.Select("next")
	routed.bindInspectorSource(s.InspectorBinding(), "next")
	if routed.pendingPick != nil || routed.navigation.Picked != nil || routed.navigation.Target != nil {
		t.Fatal("old binding request survived a session change")
	}
	routed.pendingPick = &chat.InspectorTarget{ID: "late"}
	routed.pendingPickBinding = "old"
	routed.applyPendingPick([]session.InspectorRecord{{Key: "chat:late"}})
	if routed.pendingPick != nil || routed.navigation.Picked != nil {
		t.Fatal("stale queued request resolved against a new owner")
	}
	routed.pendingPick = &chat.InspectorTarget{ID: "next"}
	routed.changeInspectorMode("session-log", "next")
	if routed.pendingPick != nil {
		t.Fatal("mode change retained a queued pick")
	}
}
