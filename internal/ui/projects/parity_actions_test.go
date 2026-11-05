package projects

import (
	"github.com/egoist/mygo/ui"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestParityNativeProjectEntryAndHistoricalActions(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	created := 0
	m := Model{Listing: projectflow.Listing{Available: true}, Workspace: "fixture", OnCreate: func() { created++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 700, 700)
	captureParityUI(t, tt, "first-project-entry.png")
	if err := tt.Click("Add Project…"); err != nil {
		t.Fatal("first-create entry was hidden", err)
	}
	if created != 1 {
		t.Fatal("first-create action was not consumed")
	}
	file := projectflow.Artifact{Path: "result.md", WorkstreamID: "work", DelegationRef: "old"}
	terminal := projectflow.DeliveryTerminalRef{WorkstreamID: "work", DelegationRef: "old", RunID: "run"}
	opened, located := 0, 0
	m.OnOpenAuthorizedFile = func(target projectflow.FileTarget) {
		if target.Root != "worker-root" {
			t.Error("root lost", target)
		}
		opened++
	}
	m.OnDeliveryFile = func(got projectflow.Artifact, open func(projectflow.FileTarget)) {
		if got != file {
			t.Error("artifact changed")
		}
		open(projectflow.FileTarget{Root: "worker-root"})
	}
	m.OnDeliveryTerminal = func(got projectflow.DeliveryTerminalRef) {
		if got != terminal {
			t.Error("settlement changed")
		}
		located++
	}
	delivery := projectflow.HistoricalDelivery{Artifacts: []projectflow.Artifact{file}, TerminalRefs: []projectflow.DeliveryTerminalRef{terminal}}
	tt = ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { deliveryRecordView(c, m, delivery) }) }, 800, 800)
	captureParityUI(t, tt, "historical-file-and-process.png")
	if err := tt.Click("Open referenced file: result.md"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Locate recorded execution: work · old · run"); err != nil {
		t.Fatal(err)
	}
	if opened != 1 || located != 1 {
		t.Fatal("historical text has no production action", opened, located)
	}
}

func TestParityHeroUsesSessionDirectoryAndRefusal(t *testing.T) {
	opened := 0
	m := Model{Listing: projectflow.Listing{Available: true}, Workspace: "/unrelated-config-directory", HeroWorkspace: "/selected", OnCreateFromDraft: func() { opened++ }}
	snap := session.ViewState{SelectedID: "blank", SelectedCwd: "/selected", ScratchCwd: "/scratch"}
	tt := ui.NewTester(func(c *ui.Context) { HeroOption(c, m, snap) }, 600, 200)
	captureParityUI(t, tt, "hero-project-option.png")
	if err := tt.Click("Project"); err != nil || opened != 1 {
		t.Fatal("selected directory Hero action missing", err, opened)
	}
	m.HeroDirectoryReason = "home"
	tt = ui.NewTester(func(c *ui.Context) { HeroOption(c, m, snap) }, 600, 200)
	if !tt.HasText("Project") {
		t.Fatal("refused directory hid the explanatory disabled option")
	}
	_ = tt.Click("Project")
	if opened != 1 {
		t.Fatal("refused Project directory was accepted")
	}
	m.HeroDirectoryReason = ""
	snap.SelectedCwd = snap.ScratchCwd
	tt = ui.NewTester(func(c *ui.Context) { HeroOption(c, m, snap) }, 600, 200)
	if tt.HasText("Project") {
		t.Fatal("scratch session exposed a Project start")
	}
	snap.SelectedCwd = "/selected"
	m.Listing.Available = false
	tt = ui.NewTester(func(c *ui.Context) { HeroOption(c, m, snap) }, 600, 200)
	if tt.HasText("Project") {
		t.Fatal("unavailable Project plugin exposed its Hero entry")
	}
}

func TestParityProjectOwnedComposerAndRules(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	project := projectflow.Project{ID: "p", CoordinatorSessionID: "lead", Lifecycle: "archived"}
	restored := 0
	m := Model{Listing: projectflow.Listing{Available: true, Projects: []projectflow.Project{project}}, OnRestoreProject: func(id string) {
		if id != "p" {
			t.Error(id)
		}
		restored++
	}}
	owned := false
	tt := ui.NewTester(func(c *ui.Context) { owned = ConversationComposer(c, m, "lead", "") }, 700, 400)
	captureParityUI(t, tt, "archived-project-composer.png")
	if !owned {
		t.Fatal("archived Lead lost its composer owner")
	}
	if err := tt.Click("Restore Project"); err != nil {
		t.Fatal(err)
	}
	if restored != 1 {
		t.Fatal("restore action missing")
	}
	project.Lifecycle = "ready"
	m.Listing.Projects = []projectflow.Project{project}
	if _, known, _ := ConversationProject(m, "ordinary", ""); known {
		t.Fatal("ordinary Session captured by Project")
	}
	draft := projectflow.MaterialDraft{ProjectID: "p", Path: "charter.md", Loaded: true}
	m.Material = &draft
	m.MaterialMutable = true
	tt = ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { materialsView(c, m) }) }, 850, 900)
	captureParityUI(t, tt, "project-charter-shortcuts.png")
	if err := tt.Click("Tests"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(draft.Draft, "must include tests") {
		t.Fatal("rule shortcut did not edit the draft", draft.Draft)
	}
	if err := tt.Click("✓ Tests"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(draft.Draft, "must include tests") {
		t.Fatal("rule shortcut could not remove the original rule")
	}
}

func captureParityUI(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("WHALE_PARITY_CAPTURE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
