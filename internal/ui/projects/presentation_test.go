package projects

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func projectFixture() Model {
	p := projectflow.Project{ID: "project-fixture", Title: "Harbor research", Lifecycle: "ready", CoordinatorSessionID: "lead-fixture", CanonicalWorkingDirectory: "/fixture/harbor"}
	return Model{ThemeID: "deepseek.light", Loaded: true, Expanded: true, Workspace: "/fixture", Listing: projectflow.Listing{Available: true, Projects: []projectflow.Project{p}}, SelectedSession: p.CoordinatorSessionID}
}

func projectJSON(t *testing.T, body string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), target); err != nil {
		t.Fatal(err)
	}
}

func TestProjectPagesUseBoundedModalStateMatrix(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	type surface struct {
		name, marker string
		configure    func(*Model)
	}
	detail := func(m *Model) {
		m.TeamID = "project-fixture"
		m.Detail = &projectflow.WorkDetail{}
		projectJSON(t, `{"workstream":{"id":"work-fixture","projectId":"project-fixture","title":"Survey work","brief":"Complete work detail body","status":"running","workerSessionId":"member-fixture"},"worker":{"role":"worker","materialized":true,"sessionId":"member-fixture","cwd":"/fixture/member"}}`, m.Detail)
	}
	request := func(m *Model) {
		m.RequestProjectID = "project-fixture"
		m.RequestKind = "current"
		m.RequestLoaded = true
		m.RequestPageNumber = 1
		m.RequestPage = projectflow.RequestPage{Items: []projectflow.RequestCard{{ID: "request-fixture", ProjectID: "project-fixture", Title: "Survey requirement", GoalPreview: "Requirement preview", Goal: "Requirement full body", Revision: 2, Status: "completed"}}, Total: 1}
	}
	surfaces := []surface{
		{"team", "Members", func(m *Model) {
			m.TeamID = "project-fixture"
			m.Team = &projectflow.Overview{Project: m.Listing.Projects[0], Team: &projectflow.TeamProjection{Members: []projectflow.TeamMember{{ID: "member-fixture", Name: "Researcher", Description: "Survey sources", Role: "worker"}}}}
		}},
		{"team-loading", "Loading…", func(m *Model) { m.TeamID = "project-fixture"; m.Busy = true }},
		{"requests", "Survey requirement", request},
		{"request-detail", "Requirement full body", func(m *Model) { request(m); card := m.RequestPage.Items[0]; m.RequestCard = &card }},
		{"requests-empty", "No user requests", func(m *Model) { request(m); m.RequestPage.Items = nil }},
		{"material-collection", "Recovery requirement", func(m *Model) {
			m.MaterialCollection = &projectflow.MaterialCollection{ProjectID: "project-fixture", Title: "Harbor research", Loaded: true, Requests: projectflow.MaterialRequests{Items: []projectflow.RequestCard{{ID: "recover-fixture", Title: "Recovery requirement", Goal: "Recorded requirement", Revision: 2}}}}
		}},
		{"notes", "Project material editor: Notes", func(m *Model) {
			m.Material = &projectflow.MaterialDraft{ProjectID: "project-fixture", Path: "notes.md", Loaded: true, Draft: "User draft", Saved: "User draft", Generated: "Generated project history"}
			m.MaterialMutable = true
		}},
		{"preferences", "Project material editor: Preferences", func(m *Model) {
			m.Material = &projectflow.MaterialDraft{ProjectID: "project-fixture", Path: "preferences.md", Loaded: true, Draft: "Preference draft", Saved: "Preference draft"}
			m.MaterialMutable = true
		}},
		{"rules", "Project material editor: Project rules", func(m *Model) {
			m.Material = &projectflow.MaterialDraft{ProjectID: "project-fixture", Path: "charter.md", Loaded: true, Draft: "Project rule draft", Saved: "Project rule draft"}
			m.MaterialMutable = true
		}},
		{"material-read-only", "This Project is read-only", func(m *Model) {
			m.Material = &projectflow.MaterialDraft{ProjectID: "project-fixture", Path: "notes.md", Loaded: true, Draft: "Retained archived notes", Saved: "Retained archived notes"}
		}},
		{"documents-empty", "no standalone documents", func(m *Model) {
			m.Documents = &projectflow.DocumentSnapshot{Scope: projectflow.DocumentScope{ProjectID: "project-fixture"}, Title: "Harbor research", Loaded: true}
		}},
		{"document-text", "Document text", func(m *Model) {
			m.Documents = &projectflow.DocumentSnapshot{Scope: projectflow.DocumentScope{ProjectID: "project-fixture"}, Title: "Harbor research", Loaded: true, TextLoaded: true, SelectedPath: "docs/summary.md", Text: "Literal document contents"}
		}},
		{"document-error", "Read failed fixture", func(m *Model) {
			m.Documents = &projectflow.DocumentSnapshot{Scope: projectflow.DocumentScope{ProjectID: "project-fixture"}, Title: "Harbor research", Error: "Read failed fixture"}
		}},
		{"deliveries-empty", "No recorded deliveries", func(m *Model) {
			m.Deliveries = &projectflow.DeliverySnapshot{Scope: projectflow.DeliveryScope{ProjectID: "project-fixture"}, Title: "Harbor research", Loaded: true}
		}},
		{"delivery-detail", "Recorded delivery summary", func(m *Model) {
			m.Deliveries = &projectflow.DeliverySnapshot{Scope: projectflow.DeliveryScope{ProjectID: "project-fixture"}, Title: "Harbor research", Loaded: true, Selected: &projectflow.HistoricalDelivery{ID: "delivery-fixture", ProjectID: "project-fixture", RequestID: "request-fixture", RequestRevision: 2, Title: "Survey delivery", Status: "delivered", Summary: "Recorded delivery summary", Applicable: true}}
		}},
		{"reports-empty", "No recorded member reports", func(m *Model) { m.Reports = &projectflow.ReportBrowser{ProjectID: "project-fixture"} }},
		{"report-detail", "Recorded member report summary", func(m *Model) {
			m.Reports = &projectflow.ReportBrowser{ProjectID: "project-fixture", WorkID: "work-fixture", Title: "Survey work", Selected: &projectflow.Report{DelegationRef: "delegation-fixture", Summary: "Recorded member report summary", Outcome: "done"}}
		}},
		{"report-error", "Report failed fixture", func(m *Model) {
			m.Reports = &projectflow.ReportBrowser{ProjectID: "project-fixture", Error: "Report failed fixture"}
		}},
		{"work-detail", "Complete work detail body", detail},
		{"process", "Process message body", func(m *Model) {
			detail(m)
			m.Process = &projectflow.ProcessSnapshot{}
			projectJSON(t, `{"type":"snapshot","records":[{"type":"event","event":{"type":"assistant/message","seq":2,"data":{"content":[{"type":"text","text":"Process message body"}]}}}]}`, m.Process)
		}},
		{"history", "Historical report marker", func(m *Model) {
			m.Historical = &projectflow.HistoricalSnapshot{Target: projectflow.HistoricalTarget{ProjectID: "project-fixture", WorkstreamID: "work-fixture", DelegationRef: "delegation-fixture", Report: projectflow.Report{Summary: "Historical report marker"}}}
		}},
		{"diff", "diff --git", func(m *Model) {
			m.TeamID = "project-fixture"
			m.AssetDiff = &projectflow.WorkDiff{}
			projectJSON(t, `{"worker":{"cwd":"/fixture/member","branch":"fixture"},"snapshot":{"isGit":true,"diff":"diff --git a/readme.md b/readme.md\n+Fixture change"}}`, m.AssetDiff)
		}},
		{"activity", "Timeline fixture", func(m *Model) {
			m.Activity = &projectflow.ActivitySnapshot{Scope: projectflow.ActivityScope{ProjectID: "project-fixture", RequestID: "request-fixture"}, Title: "Survey requirement", Loaded: true, Filter: "all", Visible: []projectflow.ActivityEvent{{Key: "event-fixture", Title: "Timeline fixture", Kind: "assignment", At: 1}}}
		}},
	}
	for _, size := range []struct {
		name          string
		width, height int
		dark          bool
	}{{"wide", 1000, 760, false}, {"narrow", 360, 480, false}, {"narrow-dark", 360, 480, true}} {
		for _, s := range surfaces {
			t.Run(s.name+"/"+size.name, func(t *testing.T) {
				m := projectFixture()
				m.Expanded = false
				s.configure(&m)
				if size.dark {
					m.ThemeID = "deepseek.dark"
				}
				closed := 0
				close := func() { closed++ }
				m.OnCloseTeam = close
				m.OnCloseRequests = close
				m.OnCloseReports = close
				m.OnCloseDetail = close
				m.OnCloseDiff = close
				m.OnCloseMaterial = close
				m.OnDocumentAction = func(kind, _ string) {
					if kind == "close" {
						close()
					}
				}
				m.OnDeliveryAction = m.OnDocumentAction
				m.OnActivityAction = m.OnDocumentAction
				m.OnHistoricalAction = func(kind string) {
					if kind == "close" {
						close()
					}
				}
				m.OnMaterialCollectionAction = m.OnHistoricalAction
				route := currentProjectRoute(m)
				tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, size.width, size.height)
				if size.dark {
					tt.SetDark(true)
				}
				if !tt.HasText(s.marker) {
					t.Fatalf("missing route data %q: %q", s.marker, tt.Texts())
				}
				bounds, ok := tt.Find(route.title)
				if !ok || bounds.W > 760 || bounds.W > float32(size.width-32)+.1 || bounds.H > float32(size.height)*.8+.1 || bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > float32(size.width)+.1 || bounds.Y+bounds.H > float32(size.height)+.1 {
					t.Fatalf("modal outside viewport: %+v", bounds)
				}
				if s.name == "reports-empty" && bounds.H >= float32(size.height)*.8-1 {
					t.Fatalf("sparse route expanded to maximum height: %+v", bounds)
				}
				if s.name == "reports-empty" {
					empty, found := tt.Find("No recorded member reports yet.")
					if !found || empty.H <= 0 || empty.Y < bounds.Y || empty.Y+empty.H > bounds.Y+bounds.H {
						t.Fatalf("sparse route body clipped: %+v modal=%+v", empty, bounds)
					}
				}
				button, ok := tt.Find("Close")
				if !ok || button.W != 28 || button.H != 28 || button.Y < bounds.Y || button.Y+button.H > bounds.Y+bounds.H {
					t.Fatalf("header Close unavailable: %+v modal=%+v", button, bounds)
				}
				if m.Material != nil {
					before, ok := tt.Find("Save")
					if !ok || before.Y+before.H > bounds.Y+bounds.H {
						t.Fatalf("material footer clipped: %+v", before)
					}
					tt.Scroll(bounds.X+50, button.Y+button.H+30, 0, 1000)
					after, _ := tt.Find("Save")
					if before != after {
						t.Fatalf("material footer moved: %+v -> %+v", before, after)
					}
					tt.Scroll(bounds.X+50, button.Y+button.H+30, 0, -1000)
				}
				saveProjectSnapshot(t, tt, "projects-"+s.name+"-"+size.name+".png")
				if m.Material != nil && m.Material.Loaded {
					if err := tt.Click("Project material editor: " + materialLabel(m.Material.Path)); err != nil {
						t.Fatal(err)
					}
					tt.Command("selectAll")
					tt.Command("copy")
					if tt.Clipboard() != m.Material.Draft {
						t.Fatalf("editor value differs: %q", tt.Clipboard())
					}
				}
				if m.Documents != nil && m.Documents.TextLoaded {
					if err := tt.Click("Document text"); err != nil {
						t.Fatal(err)
					}
					tt.Command("selectAll")
					tt.Command("copy")
					if tt.Clipboard() != m.Documents.Text {
						t.Fatalf("document text differs: %q", tt.Clipboard())
					}
				}
				tt.Key(0, ui.KeyEscape)
				if closed != 1 {
					t.Fatalf("route close callback=%d", closed)
				}
			})
		}
	}
}

func TestCompactProjectRowsKeepMenusAndDirectoryIdentity(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	m := projectFixture()
	p := m.Listing.Projects[0]
	p.ID = "other-project"
	p.CoordinatorSessionID = "other-lead"
	p.Title = strings.Repeat("Long title ", 18)
	p.Activity.State = "summaryFailed"
	m.Listing.Projects = append(m.Listing.Projects, p)
	var directory, opened string
	m.OnProjectDirectoryFor = func(id string) { directory = id }
	m.OnOpen = func(id string) { opened = id }
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Width(260).Children(func() { View(c, m) }) }, 320, 480)
	row, ok := tt.Find(p.Title)
	if !ok || row.H != 36 || row.W > 260 {
		t.Fatalf("expanded project row: %+v", row)
	}
	status, ok := tt.Find("Summary failed")
	if !ok || status.W > 96 {
		t.Fatalf("status exceeds source cap: %+v", status)
	}
	if err := tt.RightClick(p.Title); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Open project directory"); err != nil {
		t.Fatal(err)
	}
	if directory != "other-project" || opened != "" || m.SelectedSession != "lead-fixture" {
		t.Fatal("directory menu changed navigation identity", directory, opened)
	}
	tt.Move(row.X+8, row.Y+8)
	more, ok := tt.Find("More: " + p.Title)
	if !ok || more.W != 28 || more.H != 28 {
		t.Fatalf("hover menu size=%+v", more)
	}
	if tt.HasText("Summary failed") {
		t.Fatal("status did not yield to the hover/focus menu")
	}
	saveProjectSnapshot(t, tt, "projects-sidebar-narrow.png")
}

func TestProjectMaterialTransitionKeepsDraftOnFailure(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	m := projectFixture()
	m.Material = &projectflow.MaterialDraft{ProjectID: "project-fixture", Path: "notes.md", Loaded: true, Draft: "Edited user draft", Saved: "Saved notes"}
	m.MaterialMutable = true
	closed, saves := 0, 0
	m.OnCloseMaterial = func() { closed++; m.Material = nil }
	m.OnDiscardMaterial = func() { m.Material.Draft = m.Material.Saved }
	m.OnSaveMaterial = func() { saves++; m.Busy = true }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 600, 520)
	tt.Key(0, ui.KeyEscape)
	if closed != 0 || !tt.HasText("Unsaved changes") {
		t.Fatal("dirty material closed without protection")
	}
	if err := tt.Click("Save and continue"); err != nil {
		t.Fatal(err)
	}
	m.Material.Error = "Save failed fixture"
	m.Busy = false
	tt.Frame()
	if closed != 0 || saves != 1 || m.Material.Draft != "Edited user draft" || !tt.HasText("Save failed fixture") {
		t.Fatal("save failure lost draft or closed route")
	}
	saveProjectSnapshot(t, tt, "projects-material-save-error.png")
	m.OnSaveMaterial = func() { saves++; m.Material.Saved = m.Material.Draft; m.Material.Error = "" }
	if err := tt.Click("Save and continue"); err != nil {
		t.Fatal(err)
	}
	if saves != 2 || closed != 1 || m.Material != nil {
		t.Fatalf("successful save did not finish close: saves=%d closes=%d", saves, closed)
	}
	m.Material = &projectflow.MaterialDraft{ProjectID: "project-fixture", Path: "notes.md", Loaded: true, Draft: "Different unsaved notes", Saved: "Saved notes"}
	var nextID, nextPath string
	m.OnMaterial = func(id, path string) { nextID, nextPath = id, path }
	tt.Frame()
	if err := tt.Click("Preferences"); err != nil {
		t.Fatal(err)
	}
	if nextPath != "" || !tt.HasText("Unsaved changes") {
		t.Fatal("dirty tab switch bypassed confirmation")
	}
	if err := tt.Click("Discard"); err != nil {
		t.Fatal(err)
	}
	if nextID != "project-fixture" || nextPath != "preferences.md" || m.Material.Draft != "Saved notes" {
		t.Fatal("confirmed switch changed draft/project identity", nextID, nextPath)
	}
}

func TestProjectLifecycleDialogsProtectTargetAndBusyState(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	m := projectFixture()
	draft := "Harbor renamed"
	m.RenameID = "project-fixture"
	m.RenameDraft = &draft
	var id, title string
	m.OnRename = func(project, value string) { id, title = project, value }
	m.OnCancelRename = func() { m.RenameID = "" }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 360, 480)
	if !tt.Focused("Project name") {
		t.Fatal("rename input has no initial focus")
	}
	tt.Key(0, ui.KeyEnter)
	if id != "project-fixture" || title != "Harbor renamed" {
		t.Fatal("rename changed target", id, title)
	}
	m.Busy = true
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if m.RenameID == "" {
		t.Fatal("busy rename dismissed")
	}
	m.Busy = false
	tt.Frame()
	saveProjectSnapshot(t, tt, "projects-rename-narrow.png")
	tt.Key(0, ui.KeyEscape)
	if m.RenameID != "" {
		t.Fatal("rename cancellation did not dispatch")
	}
	m.ArchiveCandidate = &m.Listing.Projects[0]
	m.ArchiveCandidate.Activity.Running = 2
	m.ArchiveCandidate.Activity.State = "running"
	confirmed := 0
	m.OnConfirmArchive = func() { confirmed++ }
	m.OnCancelArchive = func() { m.ArchiveCandidate = nil }
	tt.Frame()
	button, ok := tt.Find("Stop work and archive")
	if !ok || button.Y+button.H > 480 {
		t.Fatalf("archive confirmation clipped: %+v", button)
	}
	if confirmed != 0 {
		t.Fatal("archive executed without explicit confirmation")
	}
	saveProjectSnapshot(t, tt, "projects-archive-narrow.png")
	if err := tt.Click("Stop work and archive"); err != nil {
		t.Fatal(err)
	}
	if confirmed != 1 {
		t.Fatal("archive confirmation did not dispatch")
	}
}

// The sidebar menu owns Rename/Open directory/Archive only (fixed dsh-project
// client.js:529). ProjectHeader's Project overview opens this retained observer
// (553-563); its real task actions open the independent child modal.
func TestProjectNestedRouteReturnsToScrollAndOriginFocus(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(locale)
	m := projectFixture()
	m.ObserverHost = "fixture-host"
	requestsOpened := 0
	m.OnRequests = func(id string) {
		requestsOpened++
		m.RequestProjectID = id
		m.RequestKind = "current"
		m.RequestLoaded = true
		m.RequestPageNumber = 1
	}
	for index := 0; index < 24; index++ {
		m.RequestPage.Items = append(m.RequestPage.Items, projectflow.RequestCard{ID: fmt.Sprint(index), ProjectID: "project-fixture", Revision: 1, Title: fmt.Sprintf("Requirement %02d", index), GoalPreview: strings.Repeat("Long requirement preview. ", 4), Status: "discussing"})
	}
	m.OnRequestGet = func(id string) {
		for _, card := range m.RequestPage.Items {
			if card.ID == id {
				copy := card
				m.RequestCard = &copy
				return
			}
		}
	}
	m.RequestCardActions = func(id string, revision int) RequestCardActions {
		return RequestCardActions{Known: id == "23" && revision == 1, Activity: func() {
			if id != "23" || revision != 1 {
				t.Fatal("activity changed request owner")
			}
			m.Activity = &projectflow.ActivitySnapshot{Scope: projectflow.ActivityScope{ProjectID: "project-fixture", RequestID: id}, Loaded: true, Title: "Requirement 23"}
		}}
	}
	m.OnActivityAction = func(kind, _ string) {
		if kind == "close" {
			m.Activity = nil
		}
	}
	observer := &ObserverState{ProjectID: "project-fixture", CoordinatorSessionID: "lead-fixture", Host: "fixture-host", Section: "tasks"}
	tt := ui.NewTester(func(c *ui.Context) { ObserverView(c, m, observer) }, 700, 600)
	tt.Frame()
	if requestsOpened != 1 || m.RequestProjectID != "project-fixture" {
		t.Fatal("observer did not load its exact Project")
	}
	tt.Scroll(690, 300, 0, 2000)
	if err := tt.Click("Expand task: Requirement 23"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Scroll(690, 300, 0, 2000)
	if err := tt.Click("Task details"); err != nil {
		t.Fatal(err)
	}
	tt.Scroll(690, 300, 0, 2000)
	if err := tt.Click("Activity"); err != nil {
		t.Fatal(err)
	}
	before, ok := tt.Find("Task card: 23")
	if !ok || before.Y <= 0 || before.Y >= 600 {
		t.Fatalf("scrolled task is not visible: %+v", before)
	}
	if err := tt.Click("Activity timeline"); err != nil {
		t.Fatal(err)
	}
	if m.Activity == nil || !tt.Focused("Close") {
		t.Fatal("real activity action did not open/focus child route")
	}
	tt.Key(0, ui.KeyEscape)
	after, ok := tt.Find("Task card: 23")
	if m.Activity != nil || !ok || before != after {
		t.Fatalf("return lost observer scroll: %+v -> %+v", before, after)
	}
	if !tt.Focused("Activity timeline") {
		t.Fatal("child close did not restore originating task action focus")
	}
	if requestsOpened != 1 || observer.ProjectID != "project-fixture" || observer.RequestModes["23"].Mode != "full" {
		t.Fatal("child close recreated observer or lost task mode")
	}
}

func TestProjectRowLifecycleCancelReturnsToOriginFocus(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(locale)
	m := projectFixture()
	draft := ""
	m.RenameDraft = &draft
	m.OnBeginRename = func(p projectflow.Project) { m.RenameID = p.ID; draft = p.Title }
	m.OnCancelRename = func() { m.RenameID = "" }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 700, 600)
	if err := tt.RightClick("Harbor research"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Rename"); err != nil {
		t.Fatal(err)
	}
	if m.RenameID != "project-fixture" || !tt.Focused("Project name") {
		t.Fatal("rename lost target or initial focus")
	}
	tt.Key(0, ui.KeyEscape)
	if m.RenameID != "" || !tt.Focused("More: Harbor research") {
		t.Fatal("cancel did not restore originating project menu focus")
	}
}
func saveProjectSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(file, tt.Image())
	closeErr := file.Close()
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
