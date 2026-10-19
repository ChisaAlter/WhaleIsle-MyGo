package projects

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
)

func observerTaskFixture(t *testing.T) (Model, *ObserverState) {
	t.Helper()
	m := projectFixture()
	m.ObserverHost = "fixture-host"
	m.RequestProjectID = "project-fixture"
	m.RequestKind = "current"
	m.RequestLoaded = true
	m.RequestMutable = true
	m.RequestPageNumber = 1
	var active, waiting, delivered projectflow.RequestCard
	projectJSON(t, `{"id":"active","projectId":"project-fixture","revision":2,"title":"Review the harbor evidence","status":"running","goal":"Complete original goal, conditions and retained evidence.","queue":{"state":"active"},"work":[{"workstreamId":"survey","delegationRef":"saved-1","title":"Survey sources","memberName":"Researcher","status":"completed","summary":"Saved source summary"},{"workstreamId":"review","delegationRef":"saved-2","title":"Review dependencies","memberName":"Reviewer","status":"blocked","blockedReason":"Waiting for source approval"},{"workstreamId":"write","delegationRef":"saved-3","title":"Write findings","memberName":"Writer","status":"queued"}],"versions":[{"revision":1,"title":"Original goal","goal":"Retained original version"}]}`, &active)
	projectJSON(t, `{"id":"waiting","projectId":"project-fixture","revision":1,"title":"Prepare the follow-up survey","status":"pending","queue":{"state":"queued","position":1}}`, &waiting)
	projectJSON(t, `{"id":"delivered","projectId":"project-fixture","revision":3,"title":"Publish the saved survey","status":"completed","queue":{"state":"active","released":true},"goal":"Preserve delivered goal","delivery":{"outcome":"completed","applicable":true,"summary":"Survey result retained with original source evidence.","summaryPreview":"Survey result retained with original source evidence.","artifacts":[{"title":"Survey report","path":"reports/survey.md","workstreamId":"old-work","delegationRef":"old-assignment"}]}}`, &delivered)
	m.RequestPage = projectflow.RequestPage{Items: []projectflow.RequestCard{active, waiting, delivered}, Total: 3}
	m.RequestQueue = &projectflow.RequestQueue{Active: &active, Queued: []projectflow.RequestQueueEntry{{ID: "waiting", Revision: 1}}}
	m.Listing.Projects[0].Queue = m.RequestQueue
	m.OnRequestGet = func(id string) {
		for _, card := range m.RequestPage.Items {
			if card.ID == id {
				copy := card
				m.RequestCard = &copy
				return
			}
		}
	}
	// Callers replace this closure after the returned model is bound; fixture data
	// is entirely synthetic and never reaches a Host.
	o := &ObserverState{ProjectID: "project-fixture", CoordinatorSessionID: "lead-fixture", Host: "fixture-host", Section: "tasks", Initialized: true}
	return m, o
}

func TestObserverTaskCardsRetainModesAndRouteActions(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	m, o := observerTaskFixture(t)
	reads := map[string]int{}
	m.OnRequestGet = func(id string) {
		reads[id]++
		for _, card := range m.RequestPage.Items {
			if card.ID == id {
				copy := card
				m.RequestCard = &copy
			}
		}
	}
	queueID, queueAction, archiveID := "", "", ""
	m.OnRequestQueue = func(id, action string) { queueID, queueAction = id, action }
	m.OnRequestArchive = func(id string, _ bool) { archiveID = id }
	tt := ui.NewTester(func(c *ui.Context) { ObserverView(c, m, o) }, 900, 800)
	tt.Frame()
	if o.RequestModes["active"].Mode != "card" || o.RequestModes["waiting"].Mode != "strip" || o.RequestModes["delivered"].Mode != "strip" {
		t.Fatalf("source initial modes lost: %+v", o.RequestModes)
	}
	if reads["active"] != 1 || !tt.HasText("Task progress") || !tt.HasText("Waiting for source approval") {
		t.Fatalf("active card missing source data: reads=%v text=%q", reads, tt.Texts())
	}
	strip, ok := tt.Find("Task card: waiting")
	if !ok || strip.H != 44 {
		t.Fatalf("strip height=%+v", strip)
	}
	if tt.HasText("Revision 1") || tt.HasText("Archive delivered request") {
		t.Fatal("strip exposes permanent legacy form/archive action")
	}
	if err := tt.Click("More task actions: Prepare the follow-up survey"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Cancel request"); err != nil {
		t.Fatal(err)
	}
	if queueID != "waiting" || queueAction != "cancel" {
		t.Fatalf("queue action identity=%s/%s", queueID, queueAction)
	}
	if err := tt.Click("More task actions: Publish the saved survey"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Archive delivered request"); err != nil {
		t.Fatal(err)
	}
	if archiveID != "delivered" {
		t.Fatal("archive action changed request identity")
	}
	if err := tt.Click("Task details"); err != nil {
		t.Fatal(err)
	}
	if o.RequestModes["active"].Mode != "full" || !tt.HasText("Complete original goal") {
		t.Fatalf("full body not embedded: %q", tt.Texts())
	}
	if _, modal := tt.Find("Requests and deliveries · Harbor research"); modal {
		t.Fatal("observer full mode opened a replacement modal")
	}
	if err := tt.Click("Activity"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Retained original version") {
		t.Fatal("full activity lost version body")
	}
	if err := tt.Click("Compact card"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Collapse task: Review the harbor evidence"); err != nil {
		t.Fatal(err)
	}
	if o.RequestModes["active"].Mode != "strip" {
		t.Fatal("collapse state not retained")
	}
	if err := tt.Click("Expand task: Review the harbor evidence"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Task details"); err != nil {
		t.Fatal(err)
	}
	if o.RequestModes["active"].Tab != "activity" || !tt.HasText("Retained original version") {
		t.Fatal("card collapse lost active detail tab")
	}
	m.RequestPage.Items[0].Status = "completed"
	m.RequestCard = nil
	tt.Frame()
	if o.RequestModes["active"].Mode != "card" || !o.RequestModes["active"].Completed {
		t.Fatalf("completion did not reset source default mode: %+v", o.RequestModes["active"])
	}
	// Removing a Host queue projection must never leave executable queue items.
	m.RequestQueue = nil
	m.Listing.Projects[0].Queue = nil
	if got := taskQueueActions(m, m.Listing.Projects[0], m.RequestPage.Items[1], nil); len(got) != 0 {
		t.Fatal("missing Host capability manufactured queue controls")
	}
}

func TestObserverTaskCardWideNarrowStateMatrix(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, size := range []struct {
		name string
		w, h int
		dark bool
	}{{"wide", 900, 800, false}, {"narrow", 360, 560, false}, {"narrow-dark", 360, 560, true}} {
		for _, mode := range []string{"strip", "card", "full"} {
			t.Run(mode+"/"+size.name, func(t *testing.T) {
				m, o := observerTaskFixture(t)
				if size.dark {
					m.ThemeID = "deepseek.dark"
				}
				m.OnRequestGet = func(id string) {
					for _, card := range m.RequestPage.Items {
						if card.ID == id {
							copy := card
							m.RequestCard = &copy
						}
					}
				}
				m.OnRequestQueue = func(string, string) {}
				o.RequestModes = map[string]RequestMode{"active": {Mode: mode, Tab: "overview"}}
				if mode == "full" {
					m.RequestPage.Items[0].Goal = strings.Repeat("Retained full request body. ", 80)
				}
				tt := ui.NewTester(func(c *ui.Context) { ObserverView(c, m, o) }, size.w, size.h)
				if size.dark {
					tt.SetDark(true)
				}
				tt.Frame()
				card, ok := tt.Find("Task card: active")
				if !ok || card.W > float32(size.w) || card.X < 0 {
					t.Fatalf("card overflow: %+v", card)
				}
				if mode == "strip" && card.H != 44 {
					t.Fatalf("strip=%+v", card)
				}
				if mode == "card" && (!tt.HasText("1 / 3 work items") || !tt.HasText("Researcher")) {
					t.Fatalf("card data missing: %q", tt.Texts())
				}
				if mode == "full" {
					body, ok := tt.Find("Task detail body: active")
					if !ok || body.H > float32(min(420, size.h-240))+.1 || body.H <= 0 {
						t.Fatalf("full scroll unbounded: %+v", body)
					}
					footer, _ := tt.Find("Compact card")
					tt.Scroll(body.X+20, body.Y+20, 0, 1500)
					after, _ := tt.Find("Compact card")
					if after != footer {
						t.Fatal("full-detail scrolling moved card footer")
					}
					tt.Scroll(body.X+20, body.Y+20, 0, -1500)
				}
				saveProjectSnapshot(t, tt, fmt.Sprintf("projects-observer-%s-%s.png", mode, size.name))
			})
		}
	}
}

func TestObserverFullCardsKeepIndependentActionsAndSerializeReads(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	m, o := observerTaskFixture(t)
	o.RequestModes = map[string]RequestMode{"active": {Mode: "full", Tab: "activity"}, "waiting": {Mode: "full", Tab: "overview"}}
	var reads []string
	m.OnRequestGet = func(id string) { reads = append(reads, id); m.Busy = true }
	tt := ui.NewTester(func(c *ui.Context) { ObserverView(c, m, o) }, 900, 900)
	if len(reads) != 1 || reads[0] != "active" {
		t.Fatalf("concurrent restored reads=%v", reads)
	}
	first := m.RequestPage.Items[0]
	m.RequestCard, m.Busy = &first, false
	tt.Frame()
	if len(reads) != 2 || reads[1] != "waiting" {
		t.Fatalf("next restored card not read after first settlement: %v", reads)
	}
	second := m.RequestPage.Items[1]
	m.RequestCard, m.Busy = &second, false
	var activity []string
	m.OnRequestActivity = func() { t.Fatal("A invoked the legacy current-B activity callback") }
	m.RequestCardActions = func(id string, revision int) RequestCardActions {
		known := id == first.ID && revision == first.Revision || id == second.ID && revision == second.Revision
		return RequestCardActions{Known: known, Activity: func() { activity = append(activity, id) }}
	}
	tt.Frame()
	if err := tt.Click("Activity timeline"); err != nil {
		t.Fatal(err)
	}
	if len(activity) != 1 || activity[0] != first.ID || len(reads) != 2 {
		t.Fatalf("B displaced A's independent action: activity=%v reads=%v", activity, reads)
	}
}

func TestObserverCardAssetsAndWorkUseExactCardActions(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	m, _ := observerTaskFixture(t)
	active, delivered := m.RequestPage.Items[0], m.RequestPage.Items[2]
	admitted := map[string]bool{}
	var opened, inspected, diffs []string
	m.OnResolvedFile = func(string) {}
	m.OnArtifact = func(projectflow.Artifact, func(string)) { t.Fatal("file fell through to last-read card callback") }
	m.OnWorkDiff = func(string) { t.Fatal("diff fell through to last-read card callback") }
	m.OnRequestWorkDetail = func(string, string) { t.Fatal("inspector fell through to last-read card callback") }
	m.RequestCardActions = func(id string, revision int) RequestCardActions {
		known := admitted[id] && (id == active.ID && revision == active.Revision || id == delivered.ID && revision == delivered.Revision)
		return RequestCardActions{Known: known,
			OpenFile:   func(file projectflow.Artifact, _ func(string)) { opened = append(opened, id+"/"+file.DelegationRef) },
			WorkDetail: func(work string) { inspected = append(inspected, id+"/"+work) },
			WorkDiff:   func(work string) { diffs = append(diffs, id+"/"+work) },
		}
	}
	m.RequestCard = &active
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { taskCardDetail(c, m, delivered, "results") })
	}, 700, 650)
	if err := tt.Click("Open file: Survey report"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 0 {
		t.Fatal("unadmitted card opened a file")
	}
	admitted[delivered.ID] = true
	tt.Frame()
	if err := tt.Click("Open file: Survey report"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "delivered/old-assignment" {
		t.Fatalf("file lost its card/report identity: %v", opened)
	}
	m.RequestCard = &delivered
	tt = ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { taskCardDetail(c, m, active, "team") }) }, 700, 900)
	if err := tt.Click("View work details"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("View current workspace diff"); err != nil {
		t.Fatal(err)
	}
	if len(inspected) != 0 || len(diffs) != 0 {
		t.Fatal("unadmitted revision reached a work action")
	}
	admitted[active.ID] = true
	tt.Frame()
	if err := tt.Click("View work details"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("View current workspace diff"); err != nil {
		t.Fatal(err)
	}
	if len(inspected) != 1 || inspected[0] != "active/survey" || len(diffs) != 1 || diffs[0] != "active/survey" {
		t.Fatalf("last-read delivered card displaced active work: inspector=%v diff=%v", inspected, diffs)
	}
}
