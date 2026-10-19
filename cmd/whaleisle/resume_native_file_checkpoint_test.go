package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/recovery"
	"whaleisle/internal/ui/files"
)

func TestResumeVisibilityCapturesNativeFilesInputWithoutAnotherPaint(t *testing.T) {
	for _, mode := range []string{"replace", "marked-only", "marked-commit", "selection-marked-only", "selection-marked-commit"} {
		t.Run(mode, func(t *testing.T) {
			s, q, tab, dataDir := checkpointFixture(t)
			root := t.TempDir()
			filename := filepath.Join(root, tab.Path)
			if err := os.WriteFile(filename, []byte(tab.Preview), 0600); err != nil {
				t.Fatal(err)
			}
			s.fileTabs.Workspaces = map[string]*files.WorkspaceTabs{root: {Open: []*files.FileTab{tab}, Active: tab.Path, AutoRefreshDisabled: true}}
			s.fileTabs.Root = root
			t.Cleanup(s.fileTabs.Dispose)
			selected, draft, problem, status := tab.Path, tab.Draft, "", ""
			editing, width, split, editMode := true, float32(800), float32(120), files.EditViewEdit
			m := files.Model{Tabs: &s.fileTabs, WorkspaceRoot: root, SelectedFile: &selected,
				Preview: &tab.Preview, EditDraft: &draft, PreviewErr: &problem, EditStatus: &status,
				Editing: &editing, PaneWidth: &width, PreviewSplit: &split, EditViewMode: &editMode}
			frames := 0
			tester := ui.NewTester(func(c *ui.Context) {
				frames++
				files.View(c, m)
			}, 800, 700)
			// Use the public Files surface, including its real owned local read.
			// RequestLine focuses its actual editor once that bounded read settles.
			deadline := time.Now().Add(3 * time.Second)
			for tester.NativeTextInputClient() == nil && time.Now().Before(deadline) {
				files.RequestLine(m, 1)
				tester.Frame()
				time.Sleep(time.Millisecond)
			}
			client := tester.NativeTextInputClient()
			if client == nil || problem != "" {
				t.Fatalf("owned Files native editor not ready: %s", problem)
			}
			clock := &checkpointTestClock{time: time.Unix(0, 0)}
			q.now, q.after = clock.now, clock.after
			if !q.due(clock.now(), func() {}) {
				t.Fatal("initial checkpoint unexpectedly throttled")
			}
			q.stage(captureCheckpoint(t, s), nil, clock.now())
			clock.advance(10 * time.Millisecond)
			before, oldDraft := frames, draft
			want := "native 文件草稿"
			if mode == "replace" {
				r := ui.TextInputRange{End: ui.UTF16Len(draft)}
				client.ReplaceText(&r, want)
			} else {
				end := ui.UTF16Len(draft)
				r := ui.TextInputRange{Start: end, End: end}
				if mode == "selection-marked-only" || mode == "selection-marked-commit" {
					r = ui.TextInputRange{Start: 1, End: 3}
				}
				client.SetMarkedText(&r, "待提交", ui.TextInputRange{Start: 3, End: 3})
				if _, marked := client.MarkedRange(); !marked {
					t.Fatal("fixture has no native marked text")
				}
				if mode == "selection-marked-only" || mode == "selection-marked-commit" {
					client.SetMarkedText(&r, "候选", ui.TextInputRange{Start: 2, End: 2})
				}
				want = oldDraft
				if mode == "marked-commit" || mode == "selection-marked-commit" {
					client.ReplaceText(nil, "中文")
					if mode == "selection-marked-commit" {
						want = oldDraft[:1] + "中文" + oldDraft[3:]
					} else {
						want += "中文"
					}
				}
			}
			if frames != before || draft != oldDraft {
				t.Fatal("fixture built the pending native text into its string Model")
			}
			if q.due(clock.now(), func() {}) {
				t.Fatal("last edit was not throttled")
			}
			// Application OnHide/OnMinimize order: capture before pausing content.
			// Do not Frame/Type/Key/SetFocused from this point through disk read.
			s.checkpointResumeOnVisibilityChange()
			s.fileTabs.Pause()
			clock.advance(resumeCheckpointDelay)
			reopened, err := recovery.Open(dataDir)
			if err != nil {
				t.Fatal(err)
			}
			var saved files.Tabs
			if err := json.Unmarshal(reopened.Snapshot().Files, &saved); err != nil {
				t.Fatal(err)
			}
			workspace := saved.Workspaces[root]
			if workspace == nil || len(workspace.Open) != 1 || workspace.Open[0].Draft != want || workspace.Open[0].Preview != "disk baseline" || frames != before {
				t.Fatal("visibility checkpoint missed committed native storage or persisted virtual preedit")
			}
			if bytes, err := os.ReadFile(filename); err != nil || string(bytes) != "disk baseline" {
				t.Fatal("recovery checkpoint wrote to the workspace file")
			}
		})
	}
}
