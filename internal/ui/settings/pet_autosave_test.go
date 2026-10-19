package settings

import (
	"reflect"
	"testing"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
)

func TestPetImmediateControlSaveKeepsImportedAndUnrelatedDrafts(t *testing.T) {
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	original := config.DesktopPet{Enabled: false, SkinID: "saved-missing-skin", Behavior: config.DefaultPetBehavior()}
	if err := store.SaveDesktopPet(original); err != nil {
		t.Fatal(err)
	}
	position := &config.PetPosition{X: 40, Y: 50}
	report := []string{"synthetic import draft"}
	draft := original.Behavior
	draft.Scale = 1.4
	draft.Activity = "active"
	draft.SelfTalk = !original.Behavior.SelfTalk
	// ready stays false: this fixture only saves temporary config, never a pet window.
	p := &pet.Companion{Store: store, Enabled: original.Enabled, SkinID: "unsubmitted-import-skin", Neural: true, Behavior: draft, PositionDraft: position, OriginalImportReport: report}
	savePetSetting(p, nil, "scale")
	got := store.DesktopPet()
	if p.Error != "" {
		t.Fatal(p.Error)
	}
	want := original
	want.Behavior.Scale = 1.4
	want.PersonalityConfigured = true
	if !reflect.DeepEqual(got, want) {
		t.Fatal("immediate scale save included unrelated/imported fields", got, want)
	}
	if p.SkinID != "unsubmitted-import-skin" || !p.Neural || p.PositionDraft != position || !reflect.DeepEqual(p.OriginalImportReport, report) || !reflect.DeepEqual(p.Behavior, draft) {
		t.Fatal("immediate save erased the pending native import draft", p)
	}
}
