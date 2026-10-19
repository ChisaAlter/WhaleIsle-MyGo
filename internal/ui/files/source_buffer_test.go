package files

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func nativeDraftFixture(t *testing.T, initial string) (Model, *ui.Tester, *int) {
	t.Helper()
	m := previewViewModel(initial)
	// Production frame state uses a separate current Model mirror.
	draft := initial
	m.EditDraft = &draft
	frames := new(int)
	tester := ui.NewTester(func(c *ui.Context) {
		*frames++
		ui.Column(c).Fill().Children(func() {
			sourceTextArea(c, m, m.EditDraft, false, "editor", ui.RGB(0, 0, 0)).AutoFocus()
		})
	}, 320, 180)
	if tester.NativeTextInputClient() == nil {
		t.Fatal("Files native editable client is not focused")
	}
	return m, tester, frames
}

func recoveredNativeTab(t *testing.T, raw []byte, root, path string) *FileTab {
	t.Helper()
	var restored Tabs
	if err := restored.Restore(raw); err != nil {
		t.Fatal(err)
	}
	tab := findTab(restored.Workspaces[filepath.Clean(root)], path)
	if tab == nil {
		t.Fatal("recovery lost the file owner")
	}
	if tab.editBuffer != nil {
		t.Fatal("runtime native buffer leaked into recovery")
	}
	return tab
}

func TestFilesNativeDraftExportBeforeAnotherFrame(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "primary", true: "nested-pane"}[nested], func(t *testing.T) {
			m, tester, frames := nativeDraftFixture(t, "disk baseline")
			before := *frames
			r := ui.TextInputRange{End: ui.UTF16Len(*m.EditDraft)}
			tester.NativeTextInputClient().ReplaceText(&r, "未绘制的已提交草稿")
			if *m.EditDraft != "disk baseline" || *frames != before {
				t.Fatal("fixture consumed the native input in a later build")
			}
			var raw []byte
			var err error
			if nested {
				var payload []byte
				payload, err = json.Marshal(struct{ Tabs *Tabs }{m.Tabs})
				if err == nil {
					var pane struct{ Tabs json.RawMessage }
					err = json.Unmarshal(payload, &pane)
					raw = pane.Tabs
				}
			} else {
				raw, err = m.Tabs.Export()
			}
			if err != nil {
				t.Fatal(err)
			}
			tab := recoveredNativeTab(t, raw, m.WorkspaceRoot, *m.SelectedFile)
			if tab.Draft != "未绘制的已提交草稿" || tab.Preview != "disk baseline" || !tab.Dirty() || *frames != before {
				t.Fatal("no-paint recovery missed committed native text or changed the disk baseline")
			}
			m.Tabs.Preferences.WordWrap = true
			tester.Frame()
			if *m.EditDraft != tab.Draft {
				t.Fatal("stale Model mirror overwrote the exported native draft")
			}
			tester.Key(ui.Cmd, ui.KeyZ)
			if *m.EditDraft != "disk baseline" {
				t.Fatal("export/wrap reset native undo")
			}
		})
	}
}

func TestFilesNativeDraftIMEOnlyPersistsCommittedText(t *testing.T) {
	m, tester, frames := nativeDraftFixture(t, "base")
	client := tester.NativeTextInputClient()
	before := *frames
	end := ui.TextInputRange{Start: 4, End: 4}
	client.SetMarkedText(&end, "待提交", ui.TextInputRange{Start: 3, End: 3})
	if _, marked := client.MarkedRange(); !marked {
		t.Fatal("fixture has no real native preedit")
	}
	raw, err := m.Tabs.Export()
	if err != nil {
		t.Fatal(err)
	}
	if tab := recoveredNativeTab(t, raw, m.WorkspaceRoot, *m.SelectedFile); tab.Draft != "base" || tab.Dirty() {
		t.Fatal("virtual IME preedit was serialized as committed text")
	}
	client.ReplaceText(nil, "中文")
	raw, err = m.Tabs.Export()
	if err != nil {
		t.Fatal(err)
	}
	if tab := recoveredNativeTab(t, raw, m.WorkspaceRoot, *m.SelectedFile); tab.Draft != "base中文" || *frames != before {
		t.Fatal("native IME commit required another paint")
	}
}

func TestFilesNativeDraftIMESelectionDoesNotPersistItsTemporaryDeletion(t *testing.T) {
	m, tester, frames := nativeDraftFixture(t, "abcd")
	client := tester.NativeTextInputClient()
	selection := ui.TextInputRange{Start: 1, End: 3}
	client.SetMarkedText(&selection, "中文", ui.TextInputRange{Start: 2, End: 2})
	for _, paint := range []bool{false, true} {
		if paint {
			tester.Frame()
			client.SetMarkedText(&selection, "候选", ui.TextInputRange{Start: 2, End: 2})
		}
		before := *frames
		raw, err := m.Tabs.Export()
		if err != nil {
			t.Fatal(err)
		}
		tab := recoveredNativeTab(t, raw, m.WorkspaceRoot, *m.SelectedFile)
		if tab.Draft != "abcd" || tab.Dirty() || *frames != before {
			t.Fatal("unfinished native selection replacement persisted its temporary deletion")
		}
	}
	client.ReplaceText(nil, "中文")
	before := *frames
	raw, err := m.Tabs.Export()
	if err != nil {
		t.Fatal(err)
	}
	if tab := recoveredNativeTab(t, raw, m.WorkspaceRoot, *m.SelectedFile); tab.Draft != "a中文d" || *frames != before {
		t.Fatal("native selection replacement commit was not captured without another paint")
	}
}

func TestFilesNativeDraftExternalReplacementAndInactiveOwner(t *testing.T) {
	m, tester, _ := nativeDraftFixture(t, "base")
	first := selectedTab(m)
	client := tester.NativeTextInputClient()
	r := ui.TextInputRange{End: 4}
	client.ReplaceText(&r, "native pending")
	*m.EditDraft = "explicit replacement"
	capture(m)
	if first.Draft != "explicit replacement" || first.editBuffer.String() != first.Draft {
		t.Fatal("explicit application replacement did not supersede old native input")
	}
	// Publishing an inactive tab must never write through a retained pointer to
	// the same Model field now owned by a different selected tab.
	second := &FileTab{Path: "second.txt", Preview: "second", Draft: "second"}
	w := tabsFor(m)
	w.Open = append(w.Open, second)
	w.Active = second.Path
	restore(m, second)
	first.Draft = "inactive external replacement"
	if _, err := m.Tabs.Export(); err != nil {
		t.Fatal(err)
	}
	if *m.EditDraft != "second" || first.editBuffer.String() != first.Draft || first.Preview != "base" {
		t.Fatal("inactive export corrupted the current Model or original disk baseline")
	}
}

func TestFilesNativeDraftCloseSeesPendingInput(t *testing.T) {
	m, tester, frames := nativeDraftFixture(t, "base")
	before := *frames
	r := ui.TextInputRange{End: 4}
	tester.NativeTextInputClient().ReplaceText(&r, "pending close draft")
	path := *m.SelectedFile
	if closeTab(m, path, false) || tabsFor(m).PendingClose != path || selectedTab(m).Draft != "pending close draft" || *frames != before {
		t.Fatal("close discarded a native draft before its next build")
	}
	if !closeTab(m, path, true) || len(tabsFor(m).Open) != 0 {
		t.Fatal("explicit discard did not release the tab owner")
	}
}

func TestFilesReadonlySourceDoesNotAdoptEditableBuffer(t *testing.T) {
	m := previewViewModel("readonly")
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			sourceTextArea(c, m, m.EditDraft, true, "paged-source", ui.RGB(0, 0, 0)).AutoFocus()
		})
	}, 320, 180)
	if selectedTab(m).editBuffer != nil {
		t.Fatal("read-only/paged source acquired an editable buffer")
	}
	if client := tester.NativeTextInputClient(); client != nil {
		client.ReplaceText(nil, "must not edit")
	}
	tester.Frame()
	if *m.EditDraft != "readonly" || selectedTab(m).Dirty() {
		t.Fatal("read-only source became writable")
	}
}
