package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// The app decides whether voice is offered: the rail paints the entry from the
// handler it is given, so an unwired build never shows a dead page.
func TestVoiceSectionAppearsInFeaturesOnlyWhenWired(t *testing.T) {
	wired := Model{Features: &FeaturesAccess{}, OnSelectVoice: func() {}}
	if !railHas(wired, SectionVoice) {
		t.Fatal("voice entry missing from the rail while OnSelectVoice is wired")
	}
	if railHas(Model{Features: &FeaturesAccess{}}, SectionVoice) {
		t.Fatal("voice entry rendered without a handler")
	}
}

// Opening the page sets the mounted section, and the rail must then highlight
// it with no separate *Active flag, the way market and usage already work.
func TestEmbeddedVoiceSectionKeepsTheRailOnVoice(t *testing.T) {
	m := Model{
		Features:       &FeaturesAccess{},
		OnSelectVoice:  func() {},
		ActiveSection:  SectionVoice,
		SectionContent: func(*ui.Context) {},
	}
	if got := activeSection(m, store.ViewState{}); got != SectionVoice {
		t.Fatalf("activeSection = %q, want %q", got, SectionVoice)
	}
	if got, want := sectionLabel(SectionVoice), i18n.T(i18n.KeySettingsSectionVoice); got != want {
		t.Fatalf("sectionLabel = %q, want %q", got, want)
	}
}

func railHas(m Model, id SectionID) bool {
	for _, entry := range primaryNavigation(m, store.ViewState{}) {
		if entry.id == id {
			return true
		}
	}
	return false
}
