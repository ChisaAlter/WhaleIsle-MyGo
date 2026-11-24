package schedule

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestScheduleMemoBaseInvalidatesOnDesignThemeStoreAndLocale(t *testing.T) {
	oldLocale := i18n.Locale()
	oldPrefs := theme.CurrentPreferences()
	t.Cleanup(func() {
		i18n.SetLocale(oldLocale)
		theme.SetPreferences(oldPrefs)
		design.SetGlass(false, nil)
	})

	i18n.SetLocale(i18n.LocaleEN)
	snap := schedstore.ViewState{CatalogRevision: 1, ConnectionEpoch: 1, Origin: "host", ClientReady: true}
	base := scheduleMemoBase(1, snap).Key()

	// Locale change moves version key.
	i18n.SetLocale(i18n.LocaleZH)
	if v := scheduleMemoBase(1, snap).Key(); v == base {
		t.Fatal("memo version did not change on locale switch")
	}
	i18n.SetLocale(i18n.LocaleEN)

	// Store identity epoch change moves version key.
	if v := scheduleMemoBase(2, snap).Key(); v == base {
		t.Fatal("memo version did not change on store epoch swap")
	}

	// theme.Revision() change moves version key.
	nextPrefs := oldPrefs.Clone()
	nextPrefs.Interface.Size = oldPrefs.Interface.Size + 1
	if nextPrefs.Interface.Size > 24 {
		nextPrefs.Interface.Size = 14
	}
	theme.SetPreferences(nextPrefs)
	if v := scheduleMemoBase(1, snap).Key(); v == base {
		t.Fatal("memo version did not change on theme.Revision()")
	}
	theme.SetPreferences(oldPrefs)

	// design.Revision() change moves version key.
	base2 := scheduleMemoBase(1, snap).Key()
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Card = p.Background
	})
	if v := scheduleMemoBase(1, snap).Key(); v == base2 {
		t.Fatal("memo version did not change on design.Revision()")
	}
}

func TestStandaloneViewEditAndHistoryWithCoss(t *testing.T) {
	f := newScheduleUIFixture(t)
	var editID, editTitle, editPrompt string
	var savedTitle, savedPrompt, copied string
	m := Model{
		Store:         f.store,
		SessionID:     "viewer",
		EditID:        &editID,
		EditTitle:     &editTitle,
		EditPrompt:    &editPrompt,
		HistoryText:   `{"records":[]}`,
		OnSaveTitle:   func(id, title string) { savedTitle = id + ":" + title },
		OnSavePrompt:  func(id, prompt string) { savedPrompt = id + ":" + prompt },
		OnCopyHistory: func(text string) { copied = text },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, m) })
	}, 900, 700)

	if !tt.HasText("Morning check") {
		t.Fatalf("standalone view missing task: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyScheduleEditTitle)); err != nil {
		t.Fatal(err)
	}
	if editID != "first" || editTitle != "Morning check" {
		t.Fatalf("edit did not seed row: id=%q title=%q", editID, editTitle)
	}
	if err := tt.Click(i18n.T(i18n.KeyScheduleSaveTitle)); err != nil {
		t.Fatal(err)
	}
	if savedTitle != "first:Morning check" {
		t.Fatalf("savedTitle = %q", savedTitle)
	}
	if err := tt.Click(i18n.T(i18n.KeyScheduleSavePrompt)); err != nil {
		t.Fatal(err)
	}
	if savedPrompt != "first:" {
		t.Fatalf("savedPrompt = %q", savedPrompt)
	}
	if err := tt.Click(i18n.T(i18n.KeyScheduleCopyHistory)); err != nil {
		t.Fatal(err)
	}
	if copied != `{"records":[]}` {
		t.Fatalf("copied = %q", copied)
	}
}

func TestTimingPickersCossControls(t *testing.T) {
	f := newScheduleUIFixture(t)
	p, m := f.panel()
	tt := ui.NewTester(func(c *ui.Context) { p.View(c, m) }, 1000, 700)
	if err := tt.Click("Open task: Morning check"); err != nil {
		t.Fatal(err)
	}
	// Test interval stepper buttons.
	if err := tt.Click("Increase interval"); err != nil {
		t.Fatal(err)
	}
	if d := p.drafts[p.selected]; d.Seconds != "660" {
		t.Fatalf("after increase, Seconds = %q, want 660", d.Seconds)
	}
	// Switch repeat choice via coss.Select to Daily to test clock & zone pickers.
	if err := tt.Click("Every N minutes"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Every day"); err != nil {
		t.Fatal(err)
	}
	d := p.drafts[p.selected]
	if d.Kind != "daily" {
		t.Fatalf("Kind = %q, want daily", d.Kind)
	}
	// Open clock picker and pick an adjacent hour via keyboard navigation, then another hour via click.
	if len(d.Clock) < 2 {
		t.Fatalf("unexpected initial Clock %q", d.Clock)
	}
	beforeHour, err := strconv.Atoi(d.Clock[:2])
	if err != nil {
		t.Fatalf("parse initial Clock %q: %v", d.Clock, err)
	}
	stepKey := ui.KeyDown
	wantHourNum := beforeHour + 1
	if beforeHour == 23 {
		stepKey = ui.KeyUp
		wantHourNum = 22
	}
	wantHour := fmt.Sprintf("%02d", wantHourNum)
	if err := tt.Click("Time"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, stepKey)
	tt.Key(0, ui.KeyEnter)
	if d.Clock[:2] != wantHour {
		t.Fatalf("Clock = %q, want %s:...", d.Clock, wantHour)
	}
	clickHour := fmt.Sprintf("%02d", beforeHour)
	if err := tt.Click("Hour " + clickHour); err != nil {
		t.Fatal(err)
	}
	if d.Clock[:2] != clickHour {
		t.Fatalf("Clock = %q, want %s:...", d.Clock, clickHour)
	}
	// Close clock popup via Escape and open zone picker.
	tt.Key(0, ui.KeyEscape)
	if err := tt.Click("Time zone"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Search by UTC offset, IANA ID, or city"); err != nil {
		t.Fatal(err)
	}
	tt.Type("Tokyo")
	tt.SetTime(time.Now())
	if !tt.HasText("Japan Standard Time") {
		t.Fatalf("zone search missing Japan Standard Time: %v", tt.Texts())
	}
	if err := tt.Click("UTC+09:00 · Japan Standard Time"); err != nil {
		t.Fatal(err)
	}
	if d.Zone != "Asia/Tokyo" {
		t.Fatalf("Zone = %q, want Asia/Tokyo", d.Zone)
	}
}

func TestDeliveryHistoryMemoUpdatesOnTimezoneChange(t *testing.T) {
	f := newScheduleUIFixture(t)
	i18n.SetLocale(i18n.LocaleZH)
	p, m := f.panel()
	row := f.store.Snapshot().Value.Rows[0]
	row.TimeZone = "UTC"
	p.historyKey = taskKey(row)
	p.historyRow = row
	p.historyState = deliveryHistoryState{
		Loaded: true,
		Page: schedstore.DeliveryPage{
			Records: []schedstore.DeliveryRecord{
				{MessageID: "m1", ScheduledAt: "2026-10-16T01:00:00Z"},
			},
		},
	}

	tt := ui.NewTester(func(c *ui.Context) { p.deliveryHistoryView(c, m, 1) }, 400, 300)
	if !tt.HasText("10月16日 01:00") {
		t.Fatalf("expected UTC time 01:00, got %v", tt.Texts())
	}

	p.historyRow.TimeZone = "Asia/Shanghai"
	tt.Frame()
	if !tt.HasText("10月16日 09:00") {
		t.Fatalf("expected Shanghai time 09:00 after timezone change, got %v", tt.Texts())
	}
}

func TestStandaloneListMemoUpdatesWhenRowContentChanges(t *testing.T) {
	f := newScheduleUIFixture(t)
	m := Model{Store: f.store, SessionID: "viewer"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, m) })
	}, 900, 700)
	if !tt.HasText("Morning check") {
		t.Fatalf("initial task missing: %v", tt.Texts())
	}
	f.mu.Lock()
	f.rows[0]["title"] = "Morning check v2"
	f.mu.Unlock()
	f.store.Refresh(t.Context(), "viewer")
	f.wait(t, func() bool {
		snap := f.store.Snapshot().Value
		return !snap.Loading && len(snap.Rows) > 0 && snap.Rows[0].Title == "Morning check v2"
	})
	tt.Frame()
	if !tt.HasText("Morning check v2") {
		t.Fatalf("standalone list did not update title: %v", tt.Texts())
	}
}
