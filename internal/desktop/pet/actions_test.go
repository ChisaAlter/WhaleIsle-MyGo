package pet

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/config"
)

func TestHidePersistsOnlyEnabled(t *testing.T) {
	dir := t.TempDir()
	s, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDesktopPet(config.DesktopPet{Enabled: true, SkinID: "saved", Neural: true}); err != nil {
		t.Fatal(err)
	}
	p := &Companion{Store: s, Enabled: true, SkinID: "unsaved"}
	p.hide()
	if p.Enabled || p.Error != "" {
		t.Fatalf("hide: %+v", p)
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.DesktopPet()
	if got.Enabled || got.SkinID != "saved" || !got.Neural {
		t.Fatalf("saved settings: %+v", got)
	}
	if p.SkinID != "unsaved" {
		t.Fatal("hide lost another settings draft")
	}
}

func TestHideSaveFailureKeepsPetEnabled(t *testing.T) {
	dir := t.TempDir()
	s, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDesktopPet(config.DesktopPet{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	p := &Companion{Store: s, Enabled: true}
	p.hide()
	if !p.Enabled || p.Error == "" || p.feedMessage == "" {
		t.Fatal("failed hide must stay enabled and show an error")
	}
}

func TestPetNavigationActionsAreExact(t *testing.T) {
	for _, action := range []string{"hide", "settings"} {
		valid := "whale-pet://localhost/" + action
		if taskAction(valid, TaskStatus{}) != action {
			t.Fatalf("missing %s", action)
		}
		for _, raw := range []string{valid + "?", valid + "?x=1", valid + "#x", valid + "/", "https://localhost/" + action, "whale-pet://other/" + action} {
			if taskAction(raw, TaskStatus{}) != "" {
				t.Fatalf("accepted %q", raw)
			}
		}
	}
}
