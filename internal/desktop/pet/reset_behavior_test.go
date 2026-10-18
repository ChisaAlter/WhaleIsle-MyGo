package pet

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/config"
)

func TestResetPetAppearanceBehavior(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)
	legacy := `{"pet":{"enabled":true},"live2dPet":{"enabled":true,"x":42,"y":-18,"stats":{"affection":28},"growth":{"points":999},"settings":{"lookModel":"saved-vision"}},"unrelated":"keep"}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	modified := config.PetBehavior{Scale: 1.6, Opacity: .3, Personality: "poison", Activity: "active", LockPosition: true, ShiftToDrag: true, PowerSave: true, ClickSound: true}
	saved := config.DesktopPet{Enabled: true, SkinID: "saved-skin", Neural: true, Behavior: modified, Position: &config.PetPosition{X: -120, Y: 250}}
	if err := s.SaveDesktopPet(saved); err != nil {
		t.Fatal(err)
	}
	draft := modified
	draft.Personality = "genki"
	p := &Companion{Store: s, Enabled: false, SkinID: "unsaved-skin", Neural: false, Behavior: draft}
	var mirrored []string
	p.OnPersonality = func(value string) { mirrored = append(mirrored, value) }
	if !p.ResetAppearanceBehavior() || p.Error != "" {
		t.Fatalf("reset failed: %s", p.Error)
	}
	want := config.PetBehavior{Scale: 1, Opacity: 1, Personality: "natural", Activity: "balanced", SelfTalk: true, Wander: true, ChatEnabled: true}
	if p.Behavior != want {
		t.Fatalf("draft did not receive all 11 defaults: %+v", p.Behavior)
	}
	if p.Enabled || p.SkinID != "unsaved-skin" || p.Neural {
		t.Fatal("reset changed unrelated UI drafts")
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.DesktopPet()
	if got.Behavior != want || !got.Enabled || got.SkinID != saved.SkinID || !got.Neural || got.Position == nil || *got.Position != *saved.Position || !got.PersonalityConfigured {
		t.Fatalf("reset changed fields outside behavior or did not persist defaults: %+v", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]json.RawMessage
	if err := json.Unmarshal([]byte(legacy), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatal(err)
	}
	for key, value := range before {
		var a, b bytes.Buffer
		if err := json.Compact(&a, value); err != nil {
			t.Fatal(err)
		}
		if err := json.Compact(&b, after[key]); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a.Bytes(), b.Bytes()) {
			t.Fatalf("reset changed %s", key)
		}
	}
	if len(mirrored) != 1 || mirrored[0] != "natural" {
		t.Fatalf("personality mirror: %v", mirrored)
	}
	if !p.ResetAppearanceBehavior() || len(mirrored) != 1 {
		t.Fatal("repeat reset changed personality again")
	}

	// A competing writer invalidates the store; neither committed behavior nor
	// the current editing draft may be replaced on a failed reset.
	if err := reopened.Update(func(c *config.Config) { c.Port = 4001 }); err != nil {
		t.Fatal(err)
	}
	p.Behavior = draft
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.ResetAppearanceBehavior() || p.Error == "" || p.Behavior != draft || len(mirrored) != 1 {
		t.Fatal("conflict changed the draft or published reset")
	}
	still, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, still) {
		t.Fatal("conflict overwrote the competing write")
	}
	if p.Enabled || p.SkinID != "unsaved-skin" || p.Neural {
		t.Fatal("conflict changed unrelated drafts")
	}
	p.Store = nil
	if p.ResetAppearanceBehavior() || p.Behavior != draft {
		t.Fatal("unavailable store changed the draft")
	}
}
