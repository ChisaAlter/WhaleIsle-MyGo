package sidebar

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestPinnedSidebarVisibilityAndActions(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(prev)
	m := Model{ThemeID: "deepseek.light", ViewportW: 280, ViewportH: 720, ConnectionReady: true,
		Snapshot: session.ViewState{WorkspaceReady: true, ScratchCwd: "/scratch",
			Workspaces: []session.WorkspaceRow{
				{WorkspaceID: "alpha", Title: "鲸屿桌面", Path: "/alpha", SessionIDs: []string{"a", "d", "e", "f", "normal"}},
				{WorkspaceID: "beta", Title: "移动客户端", Path: "/beta", SessionIDs: []string{"b"}},
			},
			Items: []session.Item{
				{SessionID: "a", Title: "会话恢复与历史记录", Pinned: true}, {SessionID: "b", Title: "手机远程访问", Pinned: true},
				{SessionID: "scratch", Title: "临时灵感整理", Cwd: "/scratch", Pinned: true},
				{SessionID: "d", Title: "插件兼容检查", Pinned: true}, {SessionID: "e", Title: "更新与安装", Pinned: true}, {SessionID: "f", Title: "团队任务", Pinned: true},
				{SessionID: "normal", Title: "普通工作区会话"},
			},
			PinnedSessionIDs: []string{"b", "scratch", "a", "d", "e", "f"},
			WorkspaceLayout:  session.WorkspaceLayoutState{GroupBy: "workspace", Collapsed: map[string]bool{"alpha": true, "beta": true}},
		},
	}
	var selected string
	m.OnSelect = func(id string) { selected = id }
	m.OnToggleWorkspaceGroup = func(id string) {
		m.Snapshot.WorkspaceLayout.Collapsed[id] = !session.WorkspaceGroupCollapsed(m.Snapshot, id)
	}
	m.OnMoveWorkspaceSessionBefore = func(string, string, string) { t.Fatal("pinned row used workspace reorder") }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 720)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	if !tt.HasText("已置顶") || tt.HasText("普通工作区会话") {
		t.Fatal("wrong section visibility", tt.Texts())
	}
	lastY := float32(-1)
	for _, id := range m.Snapshot.PinnedSessionIDs {
		item := m.Snapshot.Items[slices.IndexFunc(m.Snapshot.Items, func(it session.Item) bool { return it.SessionID == id })]
		r, ok := tt.Find(item.Title)
		if !ok || r.Y <= lastY {
			t.Fatalf("missing/out-of-order %s: %+v", id, r)
		}
		lastY = r.Y
		if _, ok := tt.Find("Workspace " + id); !ok {
			t.Fatalf("missing workspace context for %s", id)
		}
	}
	if targets := VisibleBatchTargets(m); len(targets) != 6 {
		t.Fatalf("select visible includes wrong rows: %+v", targets)
	}
	if err := tt.Click("手机远程访问"); err != nil || selected != "b" {
		t.Fatalf("select=%s err=%v", selected, err)
	}
	if err := tt.Click("已置顶"); err != nil {
		t.Fatal(err)
	}
	if targets := VisibleBatchTargets(m); len(targets) != 0 {
		t.Fatalf("collapsed rows batch-selectable: %+v", targets)
	}
	revealSessionSearchResult(m, "b")
	tt.Move(279, 719)
	tt.Frame()
	if m.Snapshot.WorkspaceLayout.Collapsed[session.PinnedWorkspaceGroup] || !m.Snapshot.WorkspaceLayout.Collapsed["beta"] {
		t.Fatal("search should reveal pins without opening the original workspace")
	}
	for _, variant := range []struct {
		name, theme string
		width       int
		dark        bool
	}{
		{"pinned-light", "deepseek.light", 280, false}, {"pinned-narrow-dark", "deepseek.dark", 220, true},
	} {
		m.ThemeID, m.SystemDark, m.ViewportW = variant.theme, variant.dark, float32(variant.width)
		tt.SetDark(variant.dark)
		tt.SetSize(variant.width, 720)
		tt.Frame()
		if root := os.Getenv("WHALE_ALIGNMENT_CAPTURE_DIR"); root != "" {
			if err := os.MkdirAll(root, 0755); err != nil {
				t.Fatal(err)
			}
			f, err := os.Create(filepath.Join(root, variant.name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, tt.Image()); err != nil {
				t.Fatal(err)
			}
			f.Close()
		}
	}
	// The Host pin update returns the same row to its account, without changing membership.
	m.Snapshot.PinnedSessionIDs = slices.Delete(slices.Clone(m.Snapshot.PinnedSessionIDs), 0, 1)
	m.Snapshot.Items[1].Pinned = false
	m.Snapshot.WorkspaceLayout.Collapsed["beta"] = false
	tt.Frame()
	groups := visibleWorkspaceGroups(m, m.Snapshot.Items)
	count := 0
	for _, group := range groups {
		for _, item := range group.Items {
			if item.SessionID == "b" {
				count++
				if group.Key != "beta" {
					t.Fatal("unpin did not restore account")
				}
			}
		}
	}
	if count != 1 {
		t.Fatal(fmt.Sprintf("unpin produced %d copies", count))
	}
}
