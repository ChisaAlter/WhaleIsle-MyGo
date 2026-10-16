package pet

import (
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/config"
)

func TestOriginalPetImportDraftThenSave(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", filepath.Join(dir, "codex"))
	raw := []byte(`{"live2dPet":{"enabled":true,"x":-15,"y":20,"settings":{"scale":1.3}}}`)
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := &Companion{Store: s, Behavior: config.DefaultPetBehavior()}
	p.LoadOriginalDraft()
	if p.Enabled || p.Neural || p.PositionDraft == nil || p.PositionDraft.X != -15 || p.Behavior.Scale != 1.3 || len(p.OriginalImportReport) == 0 {
		t.Fatalf("draft not imported safely: %+v", p)
	}
	saved, _ := os.ReadFile(path)
	if string(saved) != string(raw) {
		t.Fatal("load wrote config")
	}
	p.Save()
	got := s.DesktopPet()
	if p.Error != "" || got.Enabled || got.Neural || got.Position == nil || got.Position.X != -15 || got.Behavior.Scale != 1.3 || p.PositionDraft != nil || len(p.OriginalImportReport) != 0 {
		t.Fatalf("save failed: %+v error=%s", got, p.Error)
	}
}

func TestOriginalPetSkinManifestIDs(t *testing.T) {
	dir := t.TempDir()
	for name, manifest := range map[string]string{"folder": `{"id":"original-id"}`, "plain": `{}`, "dup1": `{"id":"duplicate"}`, "dup2": `{"id":"duplicate"}`} {
		if err := os.Mkdir(filepath.Join(dir, name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, "pet.json"), []byte(manifest), 0600); err != nil {
			t.Fatal(err)
		}
	}
	got := originalSkinIDs(dir, []Skin{{ID: "folder"}, {ID: "plain"}, {ID: "dup1"}, {ID: "dup2"}})
	if len(got) != 2 || got["original-id"] != "folder" || got["plain"] != "plain" {
		t.Fatalf("ID mapping=%v", got)
	}
}
