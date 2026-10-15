package pet

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"whaleisle/internal/config"
)

func TestDesktopTrayPetTogglePreservesSavedSettingsAndDraft(t *testing.T) {
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	saved := store.DesktopPet()
	saved.SkinID = "saved"
	saved.Behavior.Personality = "genki"
	if err = store.SaveDesktopPet(saved); err != nil {
		t.Fatal(err)
	}
	p := &Companion{Store: store, SkinID: "unsaved draft", Behavior: config.DefaultPetBehavior()}
	calls := 0
	p.OnEnabledChanged = func() { calls++ }
	if err = p.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	got := store.DesktopPet()
	saved.Enabled = true
	if !reflect.DeepEqual(got, saved) {
		t.Fatalf("persisted unrelated drafts: %#v", got)
	}
	if p.SkinID != "unsaved draft" || p.Behavior.Personality != config.DefaultPetBehavior().Personality || !p.Enabled || calls != 1 {
		t.Fatal("toggle changed unrelated draft or missed callback")
	}
	p.hide()
	if store.DesktopPet().Enabled || p.Enabled || calls != 2 {
		t.Fatal("pet's own hide did not resync tray")
	}
}

func TestDesktopTrayPetToggleSaveFailureKeepsState(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.json")
	store, err := config.Open(filepath.Dir(file))
	if err != nil {
		t.Fatal(err)
	}
	p := &Companion{Store: store}
	calls := 0
	p.OnEnabledChanged = func() { calls++ }
	if err = os.WriteFile(file, []byte(`{"external":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = p.SetEnabled(true); err == nil {
		t.Fatal("stale config was overwritten")
	}
	if p.Enabled || store.DesktopPet().Enabled || calls != 0 {
		t.Fatal("failed persistence changed active state")
	}
}
