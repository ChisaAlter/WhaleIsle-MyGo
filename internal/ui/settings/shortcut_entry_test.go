package settings

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/i18n"
)

func TestDSK10NativeEntry(t *testing.T) {
	if _, ok := reflect.TypeOf(InterfaceAccess{}).FieldByName("Shortcuts"); !ok {
		t.Fatal("Interface settings has no native shortcut preferences entry")
	}
}

// shortcutDialogFixture renders the General page's shortcut row and opens the
// reference dialog from it, as the user does.
func shortcutDialogFixture(t *testing.T, e *shortcuts.Editor, width, height int) *ui.Tester {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			bodyShortcutPreference(c, e, ui.RGB(20, 20, 20), ui.RGB(80, 80, 80))
		})
	}, width, height)
	if err := tt.Click("edit-shortcuts"); err != nil {
		t.Fatalf("shortcut row has no reachable editor button: %v", err)
	}
	for range 3 {
		tt.Frame()
	}
	return tt
}

// TestShortcutDialogUsesReferenceStructure pins the installed build's dialog:
// one 42 DIP row per command with its key badge, the reference search, and the
// footer's reset action and customized count. The old page-like editor (row per
// command with its own text field and a save button) must not come back.
func TestShortcutDialogUsesReferenceStructure(t *testing.T) {
	e := shortcuts.NewEditor(map[string]string{"settings.open": "Mod+Alt+F12"}, nil, func(map[string]string) error { return nil })
	tt := shortcutDialogFixture(t, e, 900, 800)

	for _, want := range []string{
		"快捷键",
		"切换左侧栏",
		"恢复全部默认",
		"1 项已自定义",
		shortcutDisplay("Mod+Alt+F12"),
	} {
		if !tt.HasText(want) {
			t.Fatalf("shortcut dialog is missing reference copy %q\n%v", want, tt.Texts())
		}
	}
	for _, stale := range []string{
		"应用与主页快捷键",
		"保存快捷键",
		"放弃草稿并刷新",
		"已保存：",
		"录制按键…",
		"使用默认值",
		"解绑",
	} {
		if tt.HasText(stale) {
			t.Fatalf("shortcut dialog still shows the page editor's copy %q", stale)
		}
	}
	// Every row is a compact entry, not an always-open editor: the dialog shows
	// its rows without a text field for each command.
	if err := tt.Click("关闭快捷键"); err != nil {
		t.Fatalf("shortcut dialog has no close action: %v", err)
	}
	if tt.HasText("恢复全部默认") {
		t.Fatal("shortcut dialog stayed open after its close action")
	}
}

// TestShortcutDialogRowRecordsAndPersistsOneRow covers the reference inline
// editor: clicking a row arms its recorder, and releasing the keys saves only
// that command.
func TestShortcutDialogRowRecordsAndPersistsOneRow(t *testing.T) {
	writes := []map[string]string{}
	e := shortcuts.NewEditor(nil, nil, func(patch map[string]string) error {
		copy := map[string]string{}
		for id, value := range patch {
			copy[id] = value
		}
		writes = append(writes, copy)
		return nil
	})
	*e.Draft("session.search") = "Mod+Alt+F11"
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("capture-settings.open"); !ok || !tt.HasText("松开组合键即保存。") {
		t.Fatalf("row did not open the reference recorder: %v", tt.Texts())
	}
	tt.PhysicalKeyDown("F9", ui.KeyF9, ui.Cmd, false, false, false)
	if len(writes) != 0 || e.Recording == "" {
		t.Fatal("keydown saved before the keys were released")
	}
	tt.PhysicalKeyUp("F9", ui.KeyF9, ui.Cmd, false)

	if len(writes) != 1 {
		t.Fatal("release did not save exactly once", writes, e.Err)
	}
	if len(writes[0]) != 1 || writes[0]["settings.open"] == "" {
		t.Fatalf("recording saved more than its own command: %v", writes[0])
	}
	if e.Active()["settings.open"] != writes[0]["settings.open"] || e.Recording != "" {
		t.Fatal("recording did not settle into the accepted binding", e.Active(), e.Recording)
	}
	if *e.Draft("session.search") != "Mod+Alt+F11" || e.Active()["session.search"] == "Mod+Alt+F11" {
		t.Fatal("recording committed or discarded an unrelated draft")
	}
	if !tt.HasText(shortcutDisplay(writes[0]["settings.open"])) {
		t.Fatalf("row badge did not show the new binding: %v", tt.Texts())
	}
}

// TestShortcutDialogRemoveAndRestoreDefaultPersistImmediately keeps the two
// per-command gestures the reference inline editor offers.
func TestShortcutDialogRemoveAndRestoreDefaultPersistImmediately(t *testing.T) {
	writes := []map[string]string{}
	e := shortcuts.NewEditor(map[string]string{"settings.open": "Mod+Alt+F12"}, nil, func(patch map[string]string) error {
		copy := map[string]string{}
		for id, value := range patch {
			copy[id] = value
		}
		writes = append(writes, copy)
		return nil
	})
	*e.Draft("session.search") = "Mod+Alt+F11"
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("unbind-settings.open"); err != nil {
		t.Fatal(err)
	}
	if len(writes) != 1 || e.Active()["settings.open"] != "" || *e.Draft("settings.open") != "" {
		t.Fatal("移除 did not persist immediately", writes, e.Active(), e.Err)
	}
	if !reflect.DeepEqual(writes[0], map[string]string{"settings.open": ""}) {
		t.Fatal("移除 submitted more than its own command", writes[0])
	}
	if *e.Draft("session.search") != "Mod+Alt+F11" || e.Active()["session.search"] == "Mod+Alt+F11" {
		t.Fatal("移除 committed or discarded an unrelated draft")
	}

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("default-settings.open"); err != nil {
		t.Fatal(err)
	}
	if len(writes) != 2 || *e.Draft("settings.open") != "Mod+," || e.Active()["settings.open"] != "Mod+," {
		t.Fatal("恢复默认 did not persist immediately", writes, e.Active(), e.Err)
	}
	if !reflect.DeepEqual(writes[1], map[string]string{"settings.open": "Mod+,"}) {
		t.Fatal("恢复默认 submitted more than its own command", writes[1])
	}
}

func TestShortcutDialogRowSaveFailureRetainsAcceptedBindingAndDrafts(t *testing.T) {
	for _, operation := range []struct{ button, value string }{
		{"unbind-settings.open", ""}, {"default-settings.open", "Mod+,"},
	} {
		t.Run(operation.button, func(t *testing.T) {
			writes := 0
			var submitted map[string]string
			e := shortcuts.NewEditor(map[string]string{"settings.open": "Mod+Alt+F12"}, nil, func(patch map[string]string) error {
				writes++
				submitted = patch
				return errors.New("synthetic persistence failure")
			})
			*e.Draft("session.search") = "Mod+Alt+F11"
			tt := shortcutDialogFixture(t, e, 900, 800)

			if err := tt.Click("shortcut-settings.open"); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click(operation.button); err != nil {
				t.Fatal(err)
			}
			if writes != 1 || !reflect.DeepEqual(submitted, map[string]string{"settings.open": operation.value}) {
				t.Fatal("row gesture did not make one narrow persistence attempt", writes, submitted)
			}
			if e.Active()["settings.open"] != "Mod+Alt+F12" || *e.Draft("settings.open") != operation.value || !strings.Contains(e.Err, "synthetic persistence failure") {
				t.Fatal("failed immediate write changed accepted binding or lost candidate", e.Active(), e.Values(), e.Err)
			}
			if *e.Draft("session.search") != "Mod+Alt+F11" || e.Active()["session.search"] == "Mod+Alt+F11" {
				t.Fatal("failed row write affected another command draft")
			}
			tt.Frame()
			if writes != 1 {
				t.Fatal("failed immediate write retried without a user gesture")
			}
		})
	}
}

// TestShortcutDialogFooterResetMatchesReference covers the footer's reset
// action and its confirmation copy.
func TestShortcutDialogFooterResetMatchesReference(t *testing.T) {
	writes := 0
	e := shortcuts.NewEditor(map[string]string{"settings.open": "Mod+Alt+F12"}, nil, func(map[string]string) error {
		writes++
		return nil
	})
	tt := shortcutDialogFixture(t, e, 900, 800)

	if !tt.HasText("1 项已自定义") {
		t.Fatalf("footer does not count customized commands: %v", tt.Texts())
	}
	if err := tt.Click("reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"恢复全部默认快捷键？",
		"将恢复当前平台的默认快捷键。你修改或移除的快捷键都会恢复，其他平台不受影响。",
		"取消",
		"恢复默认",
	} {
		if !tt.HasText(want) {
			t.Fatalf("reset confirmation is missing %q\n%v", want, tt.Texts())
		}
	}
	if writes != 0 {
		t.Fatal("reset wrote before confirmation")
	}
	if err := tt.Click("cancel-reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if e.ResetPending || writes != 0 {
		t.Fatal("reset cancel wrote or kept its confirmation")
	}
	if err := tt.Click("reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("confirm-reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	if writes != 1 || e.Active()["settings.open"] != "Mod+," || e.ResetPending {
		t.Fatal("confirmed reset did not restore and persist the defaults", writes, e.Active(), e.Err)
	}
	tt.Frame()
	if tt.HasText("1 项已自定义") || shortcutModifiedCount(e) != 0 {
		t.Fatalf("footer count did not clear after restoring the defaults: %d %v", shortcutModifiedCount(e), tt.Texts())
	}
}

func TestShortcutDialogSearchAndEmptyState(t *testing.T) {
	e := shortcuts.NewEditor(nil, nil, func(map[string]string) error { return nil })
	e.Query = "settings"
	tt := shortcutDialogFixture(t, e, 900, 800)

	if _, ok := tt.Find("shortcut-terminal.toggle"); ok {
		t.Fatal("search did not filter rows")
	}
	if _, ok := tt.Find("shortcut-settings.open"); !ok {
		t.Fatalf("search dropped the matching row: %v", tt.Texts())
	}
	e.Query = "zzz-no-command"
	tt.Frame()
	if !tt.HasText("没有匹配的快捷键") {
		t.Fatalf("search empty state missing: %v", tt.Texts())
	}
}

// TestShortcutReferenceDialogSnapshot writes the production dialog render at
// the reference viewport for the visual comparison.
func TestShortcutReferenceDialogSnapshot(t *testing.T) {
	e := shortcuts.NewEditor(map[string]string{"settings.open": "Mod+Alt+F12", "session.fork": ""}, nil, func(map[string]string) error { return nil })
	tt := shortcutDialogFixture(t, e, 1440, 920)
	for range 12 {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
	if !tt.HasText("恢复全部默认") {
		t.Fatalf("snapshot did not render the reference dialog: %v", tt.Texts())
	}
	saveBodySnapshot(t, tt, "settings-shortcuts-dialog-1440.png")

	// The inline editor is the reference's second state: the row keeps its label
	// and the binding column holds 恢复默认, 移除 and the recorder.
	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	for range 4 {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
	if !tt.HasText("松开组合键即保存。") {
		t.Fatalf("snapshot did not reach the inline editor: %v", tt.Texts())
	}
	saveBodySnapshot(t, tt, "settings-shortcuts-editor-1440.png")
}
