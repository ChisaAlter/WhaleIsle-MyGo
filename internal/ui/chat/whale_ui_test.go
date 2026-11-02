package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestWhaleUIFenceDialects(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	for _, language := range []string{"whale-ui", "dsh-ui", "openui", "openui-lang"} {
		source := `root = Col([Text("Visible")])`
		tt := ui.NewTester(func(c *ui.Context) {
			if !openUIFence(c, Model{FeedbackStore: session.New(), Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "b"}}, session.Message{ID: "m", Role: "assistant"}, language, source, 14) {
				t.Error("unrecognized", language)
			}
		}, 800, 900)
		if err := tt.Click(i18n.T(i18n.KeyChatIsolatedSource)); err != nil || !tt.HasText(source) {
			t.Fatal("dialect source UI missing", language, err)
		}
	}
}
func TestWhaleUIPanelPersistenceAndRollback(t *testing.T) {
	root := t.TempDir()
	t.Setenv(paths.EnvUserData, root)
	m := Model{Snapshot: session.ViewState{SelectedID: "panel-session", PlanHistoryBinding: "binding"}}
	p := whalePanelFor(m)
	first := json.RawMessage(`{"append":false,"spec":{"title":"Fixture","items":[{"type":"text","content":"First"}]}}`)
	second := json.RawMessage(`{"append":true,"spec":{"items":[{"type":"text","content":"Second"}]}}`)
	if !strings.HasPrefix(p.Path, root) {
		t.Fatal("profile escaped")
	}
	if err := p.Publish("a", [3]float64{5, 0, 0}, first); err != nil {
		t.Fatal(err)
	}
	if err := p.Publish("b", [3]float64{6, 0, 0}, second); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(p.Snapshot), "First") || !strings.Contains(string(p.Snapshot), "Second") {
		t.Fatal("append not merged", string(p.Snapshot))
	}
	whalePanels.Lock()
	delete(whalePanels.values, p.Path)
	whalePanels.Unlock()
	reopened := whalePanelFor(m)
	if string(reopened.Snapshot) != string(p.Snapshot) {
		t.Fatal("persisted fold lost")
	}
	if err := reopened.Publish("old", [3]float64{5, 0, 0}, first); err != nil {
		t.Fatal(err)
	}
	if string(reopened.Snapshot) != string(p.Snapshot) {
		t.Fatal("replay replaced newer panel")
	}
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	oldPath := p.Path
	p.Path = filepath.Join(blocker, "cannot-write.json")
	before, generation := string(p.Snapshot), p.Generation
	if err := p.Clear(); err == nil || string(p.Snapshot) != before || p.Generation != generation {
		t.Fatal("failed clear committed")
	}
	if err := p.Publish("later", [3]float64{7, 0, 0}, first); err == nil || string(p.Snapshot) != before {
		t.Fatal("failed publish committed")
	}
	p.Path = oldPath
	if err := reopened.Clear(); err != nil {
		t.Fatal(err)
	}
	if reopened.Current() {
		t.Fatal("clear left content current")
	}
	if err := reopened.Publish("old", [3]float64{6, 0, 0}, second); err != nil {
		t.Fatal(err)
	}
	if reopened.Current() {
		t.Fatal("clear allowed old-source replay")
	}
	t.Setenv(paths.EnvUserData, t.TempDir())
	other := whalePanelFor(m)
	if other.Path == p.Path || other.Current() {
		t.Fatal("profile state leaked")
	}
}
func TestWhaleUIResolvedLargeProgramCurrentGeneration(t *testing.T) {
	for _, tc := range []struct {
		revision uint64
		program  string
		want     bool
	}{{7, "", true}, {6, "", false}, {8, "", false}, {7, "other", false}, {0, "old", false}, {0, "program", true}} {
		if got := whaleResolvedCurrent(openui.Event{Revision: tc.revision, Program: tc.program}, "program", 7); got != tc.want {
			t.Fatalf("resolved revision=%d program=%q got=%v", tc.revision, tc.program, got)
		}
	}
}
func TestWhaleUIDownloadAssemblyAndNoOverwrite(t *testing.T) {
	d := newWhaleDownloads()
	e := openui.Event{Type: "download-part", Sequence: 1, RequestID: "native-action-1", UserGesture: true, Filename: "result.html", MIME: "text/html", Parts: 2, Content: "hello ", Part: 0}
	if err := d.Add(e); err != nil {
		t.Fatal(err)
	}
	e.Part = 1
	e.Content = "world"
	if err := d.Add(e); err != nil {
		t.Fatal(err)
	}
	e.Type = "download"
	e.Part = 0
	e.Content = ""
	if err := d.Complete(&e); err != nil || e.Content != "hello world" {
		t.Fatal("assembled export", err)
	}
	path := filepath.Join(t.TempDir(), "result.html")
	if err := writeWhaleExport(path, []byte(e.Content)); err != nil {
		t.Fatal(err)
	}
	if err := writeWhaleExport(path, []byte("overwrite")); err == nil {
		t.Fatal("existing export overwritten")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "hello world" {
		t.Fatal("existing bytes changed")
	}
	d = newWhaleDownloads()
	e.Type = "download-part"
	e.Part = 1
	if err := d.Add(e); err == nil {
		t.Fatal("out of order part accepted")
	}
}
