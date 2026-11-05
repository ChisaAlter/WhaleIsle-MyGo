package sidebar

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSessionRowsStayCompactAcrossSelectionAndMenu(t *testing.T) {
	for _, width := range []int{180, 280} {
		selected := "a"
		title := strings.Repeat("Long title ", 10)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				m := Model{Snapshot: session.ViewState{SelectedID: selected}, OnArchive: func(string) {}}
				rowSession(c.Key("a"), session.Item{SessionID: "a", Title: title, Cwd: "C:/a/long/path", Running: true}, 1, m, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), ui.Transparent)
				rowSession(c.Key("b"), session.Item{SessionID: "b", Title: "Second"}, 1, m, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), ui.Transparent)
			})
		}, width, 160)
		check := func() {
			t.Helper()
			first, ok := tt.Find(title)
			second, ok2 := tt.Find("Second")
			if !ok || !ok2 || first.H != 32 || second.H != 32 || second.Y-first.Y != 32 || first.W != float32(width) {
				t.Fatalf("width %d row geometry: first=%+v second=%+v", width, first, second)
			}
			if _, found := tt.Find("C:/a/long/path"); found {
				t.Fatal("selected path expanded the row")
			}
			status, found := tt.Find(statusLabel("running"))
			if !found || status.W != 16 || status.H != 20 {
				t.Fatalf("status slot=%+v", status)
			}
		}
		check()
		if err := tt.Click(sessionActionsLabel("a", title)); err != nil {
			t.Fatal(err)
		}
		check()
		tt.CloseMenu()
		selected = "b"
		tt.Frame()
		check()
	}
}

func TestSessionMenusKeepRowIdentityAndCallbacks(t *testing.T) {
	var action, target, title string
	pinned := false
	m := Model{Snapshot: session.ViewState{SelectedID: "other"},
		OnPin:         func(id string) { action, target = "pin", id },
		OnUnpin:       func(id string) { action, target = "unpin", id },
		OnBeginRename: func(id, value string) { action, target, title = "rename", id, value },
		OnMoveSessionBefore: func(id, before string) {
			action, target = "move", id
			if before != "" {
				t.Fatal(before)
			}
		},
		OnFork:          func(id string) { action, target = "fork", id },
		OnCopySessionID: func(id string) { action, target = "copy", id },
		OnArchive:       func(id string) { action, target = "archive", id },
		OnSelect:        func(string) { t.Fatal("menu selected the conversation") },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			rowSession(c, session.Item{SessionID: "row", Title: "Original title", Pinned: pinned}, 0, m, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), ui.Transparent)
		})
	}, 280, 100)
	for _, tc := range []struct{ key, action string }{{i18n.KeySidebarPin, "pin"}, {i18n.KeySidebarRename, "rename"}, {i18n.KeySidebarFork, "fork"}, {i18n.KeySidebarArchive, "archive"}, {i18n.KeySidebarCopyID, "copy"}} {
		if err := tt.Click(sessionActionsLabel("row", "Original title")); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(i18n.T(tc.key)); err != nil {
			t.Fatal(err)
		}
		if action != tc.action || target != "row" {
			t.Fatalf("action=%s target=%s", action, target)
		}
	}
	if title != "Original title" {
		t.Fatalf("rename title=%q", title)
	}
	// The focused trigger must remain in the tree and usable without a pointer.
	for _, key := range []ui.Key{ui.KeyEnter, ui.KeySpace, ui.KeyDown} {
		tt.Key(0, key)
		if tt.Menu() == nil {
			t.Fatalf("key %v did not open menu", key)
		}
		tt.CloseMenu()
	}
	pinned = true
	tt.Frame()
	if err := tt.Click(sessionActionsLabel("row", "Original title")); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(i18n.T(i18n.KeySidebarUnpin)); err != nil {
		t.Fatal(err)
	}
	if action != "unpin" || target != "row" {
		t.Fatalf("unpin action=%s target=%s", action, target)
	}
}

func TestWorkspaceSearchIsInlineAndEscapeClears(t *testing.T) {
	query, open := "", false
	refreshed, selected := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			workspaceHeading(c, Model{SearchQuery: &query, OnRefresh: func() { refreshed++ }, OnBatchEdit: func(string, []session.SessionBatchTarget) { selected++ }}, &open, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0))
			ui.Text(c, "Below heading")
		})
	}, 280, 100)
	before, _ := tt.Find("Below heading")
	if err := tt.Click(i18n.T(i18n.KeySidebarSearch)); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused(i18n.T(i18n.KeySidebarSearch)) {
		t.Fatal("search did not focus inline input")
	}
	tt.Type("needle")
	if query != "needle" || refreshed != 0 {
		t.Fatalf("query=%q refresh=%d", query, refreshed)
	}
	after, _ := tt.Find("Below heading")
	if before.Y != after.Y {
		t.Fatalf("expansion moved rows: %v to %v", before, after)
	}
	tt.Key(0, ui.KeyEscape)
	if query != "" {
		t.Fatal("Escape did not clear")
	}
	if _, ok := tt.Find(workspaceText("清除搜索", "Clear search")); ok {
		t.Fatal("Escape left input expanded")
	}
	if err := tt.Click(workspaceText("多选会话", "Select sessions")); err != nil {
		t.Fatal(err)
	}
	if selected != 1 {
		t.Fatal("inline selection toggle missing")
	}
}

func TestAuthoritativeWorkspacesDoNotDuplicateCurrentFolder(t *testing.T) {
	for _, tc := range []struct {
		name     string
		snapshot session.ViewState
		feedback string
		rows     int
	}{
		{"disconnected", session.ViewState{}, workspaceText("连接后显示会话", "Connect to load sessions"), 0},
		{"ready-empty", session.ViewState{WorkspaceReady: true}, workspaceText("暂无会话", "No sessions yet"), 0},
		{"authoritative", session.ViewState{WorkspaceReady: true, Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", Title: "Host workspace"}}}, "Host workspace", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selected, toggled := "", ""
			menuOpen := false
			m := Model{WorkspaceLabel: "Legacy folder", Snapshot: tc.snapshot, WorkspaceMenuOpen: &menuOpen,
				OnSelectWorkspace:      func(id string) { selected = id },
				OnToggleWorkspaceGroup: func(id string) { toggled = id }}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 800)
			// The original WorkspaceBrowser renders its registry or empty state;
			// neither the disconnected nor ready-empty list synthesizes cwd.
			if _, legacy := tt.Find("Legacy folder"); legacy {
				t.Fatal("configured folder duplicated the workspace list")
			}
			if _, found := tt.Find(tc.feedback); !found {
				t.Fatalf("missing workspace state %q: %v", tc.feedback, tt.Texts())
			}
			rows := 0
			for _, text := range tt.Texts() {
				if text == workspaceText("折叠工作区 ", "Toggle workspace ")+"ws" {
					rows++
				}
			}
			if rows != tc.rows {
				t.Fatalf("Host workspace rows=%d, want %d", rows, tc.rows)
			}
			if tc.rows == 1 {
				if err := tt.Click("Host workspace"); err != nil {
					t.Fatal(err)
				}
				if selected != "" || toggled != "ws" {
					t.Fatalf("workspace click selected=%q toggled=%q", selected, toggled)
				}
			} else if _, found := tt.Find("Host workspace"); found {
				t.Fatal("non-authoritative state synthesized a Host workspace")
			}
		})
	}
}

func TestRowRenameFocusCommitAndCancel(t *testing.T) {
	draft, committed, cancelled := "Original", "", false
	m := Model{RenameSessionID: "row", RenameDraft: &draft,
		OnCommitRename: func(id, value string) {
			if id != "row" {
				t.Fatal(id)
			}
			committed = value
		},
		OnCancelRename: func() { cancelled = true },
		OnSelect:       func(string) { t.Fatal("rename selected the conversation") },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			rowSession(c, session.Item{SessionID: "row", Title: "Original"}, 0, m, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), ui.Transparent)
		})
		sessionRenameDialog(c, m)
	}, 800, 600)
	tt.Key(ui.Ctrl, ui.KeyA)
	tt.Type("Renamed")
	tt.Key(0, ui.KeyEnter)
	if committed != "Renamed" {
		t.Fatalf("committed=%q draft=%q", committed, draft)
	}
	tt.Key(0, ui.KeyEscape)
	if !cancelled {
		t.Fatal("Escape did not cancel row rename")
	}
}
