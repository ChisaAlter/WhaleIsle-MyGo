package schedule

import (
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
	// Open clock picker and pick hour 10 via keyboard navigation, then 08 via click.
	if err := tt.Click("Time"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if d.Clock[:2] != "10" {
		t.Fatalf("Clock = %q, want 10:...", d.Clock)
	}
	if err := tt.Click("Hour 08"); err != nil {
		t.Fatal(err)
	}
	if d.Clock[:2] != "08" {
		t.Fatalf("Clock = %q, want 08:...", d.Clock)
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
