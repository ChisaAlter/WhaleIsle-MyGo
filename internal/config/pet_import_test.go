package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOriginalPetDraftExplicitAndNonDestructive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := []byte(`{"pet":{"enabled":true,"petId":"cat","xRatio":0.8,"yRatio":0.2},"live2dPet":{"enabled":true,"x":-120.5,"y":0,"settings":{"scale":1.3,"opacity":0.7,"wander":false}},"mygoDesktopPet":{"enabled":false,"skinId":"keep","neural":false}}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	base := s.DesktopPet()
	got, report := s.OriginalPetDraft(base, map[string]string{"cat": "cat"})
	if got.Enabled || got.Neural || got.SkinID != "cat" || got.Position == nil || *got.Position != (PetPosition{X: -120, Y: 0}) || got.Behavior.Scale != 1.3 || got.Behavior.Wander || len(report) == 0 {
		t.Fatalf("draft = %+v, report=%v", got, report)
	}
	onDisk, _ := os.ReadFile(path)
	if string(onDisk) != string(raw) || !reflect.DeepEqual(s.DesktopPet(), base) {
		t.Fatal("draft changed saved settings")
	}
	if err := s.SaveDesktopPet(got); err != nil {
		t.Fatal(err)
	}
	var before, after map[string]json.RawMessage
	_ = json.Unmarshal(raw, &before)
	onDisk, _ = os.ReadFile(path)
	_ = json.Unmarshal(onDisk, &after)
	for _, k := range []string{"pet", "live2dPet"} {
		var a, b any
		_ = json.Unmarshal(before[k], &a)
		_ = json.Unmarshal(after[k], &b)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("changed original %s", k)
		}
	}
}

func TestOriginalPetDraftRejectsUnsupportedSources(t *testing.T) {
	for _, raw := range []string{
		`{"pet":{"petId":"missing","xRatio":0.5,"yRatio":0.5},"live2dPet":{"x":null,"y":20}}`,
		`{"live2dPet":{"x":"20","y":30,"settings":[]}}`,
		`{"live2dPet":{"x":1048577,"y":30}}`,
		`{"pet":[],"live2dPet":null}`,
	} {
		path := filepath.Join(t.TempDir(), "config.json")
		_ = os.WriteFile(path, []byte(raw), 0600)
		s, err := Open(filepath.Dir(path))
		if err != nil {
			t.Fatal(err)
		}
		base := DesktopPet{Enabled: true, Neural: true, SkinID: "keep", Behavior: DefaultPetBehavior(), Position: &PetPosition{X: 4, Y: 5}}
		got, _ := s.OriginalPetDraft(base, nil)
		if !reflect.DeepEqual(got, base) {
			t.Fatalf("unsupported source changed draft: %s -> %+v", raw, got)
		}
	}
}
