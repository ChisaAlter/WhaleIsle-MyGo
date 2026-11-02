package sidebar

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSessionRenameRootDialogPrefillsSelectsAndTrims(t *testing.T) {
	draft, got := "Original 名称", ""
	cancelled := false
	m := Model{RenameSessionID: "A", RenameDraft: &draft, Collapsed: true, OnCommitRename: func(id, title string) {
		if id != "A" {
			t.Fatal(id)
		}
		got = title
	}, OnCancelRename: func() { cancelled = true }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 600)
	tt.SetTime(time.Now().Add(300 * time.Millisecond))
	tt.Frame()
	dialog, ok := tt.Find(workspaceText("重命名会话", "Rename session"))
	if !ok || dialog.W != 380 || dialog.H != 206 || dialog.X < 200 || dialog.Y < 150 {
		t.Fatalf("rename root dialog bounds=%+v present=%t", dialog, ok)
	}
	client := tt.NativeTextInputClient()
	if client == nil || client.Selection().Range != (ui.TextInputRange{Start: 0, End: ui.UTF16Len(draft)}) {
		t.Fatal("prefill did not select the whole original title")
	}
	tt.Type("  Updated 名称  ")
	tt.Key(0, ui.KeyEnter)
	if got != "Updated 名称" {
		t.Fatalf("submitted=%q draft=%q", got, draft)
	}
	tt.Key(0, ui.KeyEscape)
	if !cancelled {
		t.Fatal("Escape failed to cancel root rename dialog")
	}
}

func TestSessionRenameBusyEmptyUnchangedAndIME(t *testing.T) {
	draft := "Original"
	commits, cancels := 0, 0
	m := Model{RenameSessionID: "A", RenameDraft: &draft, OnCommitRename: func(string, string) { commits++ }, OnCancelRename: func() { cancels++ }}
	tt := ui.NewTester(func(c *ui.Context) { sessionRenameDialog(c, m) }, 800, 600)
	tt.SetTime(time.Now().Add(300 * time.Millisecond))
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if commits != 1 {
		t.Fatal("unchanged title was incorrectly blocked")
	}
	m.RenameBusy = true
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if cancels != 0 {
		t.Fatal("busy request cancelled")
	}
	_ = tt.Click(i18n.T(i18n.KeySidebarRename))
	if commits != 1 {
		t.Fatal("busy request submitted twice")
	}
	m.RenameBusy = false
	draft = " "
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if commits != 1 {
		t.Fatal("empty rename submitted")
	}
	draft = "Original"
	tt.Frame()
	if err := tt.Click(workspaceText("会话名称", "Session name")); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused(workspaceText("会话名称", "Session name")) {
		t.Fatal("rename input not focused before IME")
	}
	tt.Compose("中文", 2)
	if _, marked := tt.NativeTextInputClient().MarkedRange(); !marked {
		t.Fatal("native composition did not reach rename editor")
	}
	tt.Key(0, ui.KeyEnter)
	if commits != 1 {
		t.Fatal("IME choosing Enter submitted a rename")
	}
	tt.Compose("", 0)
	tt.Type("中文")
	if commits != 1 {
		t.Fatal("committing composed text submitted a rename")
	}
	tt.Key(0, ui.KeyEnter)
	if commits != 1 {
		t.Fatal("the composition-finishing Enter submitted a rename")
	}
	tt.Key(0, ui.KeyEnter)
	if commits != 2 {
		t.Fatal("ordinary Enter after composition did not submit")
	}
}

// The title becomes a pointer target for double-clicking. It must retain the
// row's existing reorder contract rather than swallowing title-origin drags.
func TestSessionRenameTitleKeepsWorkspaceDrag(t *testing.T) {
	items := []session.Item{{SessionID: "a", Title: "Alpha"}, {SessionID: "b", Title: "Beta"}, {SessionID: "c", Title: "Gamma"}}
	calls := 0
	m := Model{Snapshot: session.ViewState{Items: items, Workspaces: []session.WorkspaceRow{{WorkspaceID: "ws", SessionIDs: []string{"a", "b", "c"}}}}, OnBeginRename: func(string, string) { t.Fatal("drag opened rename") }, OnSelect: func(string) { t.Fatal("drag selected conversation") }, OnMoveWorkspaceSessionBefore: func(workspace, id, before string) {
		if workspace != "ws" || id != "c" || before != "a" {
			t.Fatal(workspace, id, before)
		}
		calls++
	}}
	m.Snapshot.WorkspaceLayout.HostGeneration = 7
	m.Snapshot.WorkspaceLayout.Collapsed = map[string]bool{"ws": false}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { workspaceRowsView(c, m, items, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), ui.Transparent) })
	}, 300, 400)
	src, ok := tt.Find("Gamma")
	if !ok {
		t.Fatal("source missing")
	}
	dst, ok := tt.Find("Alpha")
	if !ok {
		t.Fatal("target missing")
	}
	tt.Press(src.X+55, src.Y+10)
	tt.Move(src.X+65, src.Y+10)
	tt.Move(dst.X+55, dst.Y+5)
	tt.Release(dst.X+55, dst.Y+5)
	if calls != 1 {
		t.Fatalf("title reorder callbacks=%d", calls)
	}
}

func TestSessionRenameDoubleClickIsTitleOnlyAndBlankRefuses(t *testing.T) {
	for _, blank := range []bool{false, true} {
		t.Run(map[bool]string{false: "named", true: "blank"}[blank], func(t *testing.T) {
			began, selections := 0, 0
			item := session.Item{SessionID: "A", Title: "Original title", Blank: blank}
			m := Model{OnBeginRename: func(id, title string) {
				if id != "A" || title != item.Title {
					t.Fatal(id, title)
				}
				began++
			}, OnSelect: func(string) { selections++ }}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Width(280).Children(func() { rowSession(c, item, 0, m, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0), ui.Transparent) })
			}, 300, 80)
			label := item.Title
			if blank {
				label = i18n.T(i18n.KeySidebarNewSessionFallback)
			}
			r, ok := tt.Find(label)
			if !ok {
				t.Fatal("row absent")
			}
			tt.ClickAt(r.X+10, r.Y+r.H/2)
			tt.ClickAt(r.X+10, r.Y+r.H/2)
			if began != 0 {
				t.Fatal("double-clicking the leading slot opened rename")
			}
			tt.ClickAt(r.X+55, r.Y+r.H/2)
			tt.ClickAt(r.X+55, r.Y+r.H/2)
			if blank && began != 0 || !blank && began != 1 {
				t.Fatalf("blank=%t begin=%d", blank, began)
			}
			if selections < 1 {
				t.Fatal("ordinary title click no longer selects")
			}
		})
	}
}
