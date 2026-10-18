package config

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPetBehaviorDefaults(t *testing.T) {
	want := PetBehavior{Scale: 1, Opacity: 1, Personality: "natural", Activity: "balanced", SelfTalk: true, Wander: true, ChatEnabled: true}
	if got := DefaultPetBehavior(); got != want {
		t.Fatalf("defaults = %+v, want %+v", got, want)
	}
	if got := NormalizePetBehavior(PetBehavior{}); got != want {
		t.Fatalf("legacy zero value = %+v, want %+v", got, want)
	}
	for _, raw := range []string{`{}`, `null`} {
		var got PetBehavior
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("decode %s = %+v, want %+v", raw, got, want)
		}
	}
}

func TestPetBehaviorDecodeDefaultsEachInvalidField(t *testing.T) {
	var got PetBehavior
	raw := `{"scale":"large","opacity":0.68,"personality":"unknown","activity":"active","selfTalk":false,"wander":"yes","chatEnabled":false,"lockPosition":true,"shiftToDrag":null,"powerSave":true,"clickSound":true}`
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	want := PetBehavior{Scale: 1, Opacity: 0.7, Personality: "natural", Activity: "active", SelfTalk: false, Wander: true, ChatEnabled: false, LockPosition: true, ShiftToDrag: false, PowerSave: true, ClickSound: true}
	if got != want {
		t.Fatalf("decode = %+v, want %+v", got, want)
	}
}

func TestPetBehaviorDecodeKeepsExplicitFalse(t *testing.T) {
	var got PetBehavior
	if err := json.Unmarshal([]byte(`{"selfTalk":false,"wander":false,"chatEnabled":false}`), &got); err != nil {
		t.Fatal(err)
	}
	if got.SelfTalk || got.Wander || got.ChatEnabled || got.Scale != 1 || got.Opacity != 1 {
		t.Fatalf("explicit false changed or defaults lost: %+v", got)
	}
	got = NormalizePetBehavior(got)
	if got.SelfTalk || got.Wander || got.ChatEnabled {
		t.Fatalf("normalization re-enabled an explicit false: %+v", got)
	}
}

func TestPetBehaviorClampAndSnap(t *testing.T) {
	cases := []struct {
		name                   string
		scale, opacity         float64
		wantScale, wantOpacity float64
	}{
		{"minimum", -3, 0, 0.6, 0.3},
		{"maximum", 9, 4, 1.6, 1},
		{"snap up", 1.26, 0.68, 1.3, 0.7},
		{"snap down", 1.24, 0.67, 1.2, 0.65},
		{"nonfinite", math.NaN(), math.Inf(1), 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := DefaultPetBehavior()
			input.Scale, input.Opacity = tc.scale, tc.opacity
			got := NormalizePetBehavior(input)
			if got.Scale != tc.wantScale || got.Opacity != tc.wantOpacity {
				t.Fatalf("normalized = %v/%v, want %v/%v", got.Scale, got.Opacity, tc.wantScale, tc.wantOpacity)
			}
		})
	}
}

func TestPetBehaviorExplicitZeroDoesNotRestoreBooleans(t *testing.T) {
	var got PetBehavior
	if err := json.Unmarshal([]byte(`{"scale":0,"opacity":0,"personality":"","activity":"","selfTalk":false,"wander":false,"chatEnabled":false}`), &got); err != nil {
		t.Fatal(err)
	}
	if got.Scale != 0.6 || got.Opacity != 0.3 || got.SelfTalk || got.Wander || got.ChatEnabled {
		t.Fatalf("explicit zero-valued preferences became legacy defaults: %+v", got)
	}
}

func TestPetBehaviorWhitelists(t *testing.T) {
	for _, personality := range []string{"natural", "genki", "tsundere", "poison"} {
		for _, activity := range []string{"quiet", "balanced", "active"} {
			input := DefaultPetBehavior()
			input.Personality, input.Activity = personality, activity
			if got := NormalizePetBehavior(input); got != input {
				t.Fatalf("valid choices changed: %+v -> %+v", input, got)
			}
		}
	}
	input := DefaultPetBehavior()
	input.Personality, input.Activity = `"><script>`, "extreme"
	if got := NormalizePetBehavior(input); got.Personality != "natural" || got.Activity != "balanced" {
		t.Fatalf("invalid choices survived: %+v", got)
	}
}

func TestDesktopPetBehaviorReadIsIndependent(t *testing.T) {
	for _, raw := range []string{
		`{"enabled":"yes","skinId":"test","neural":true,"behavior":{"opacity":"bad","selfTalk":false},"position":{"x":-100,"y":0}}`,
		`{"enabled":"yes","skinId":"test","neural":true,"behavior":false,"position":{"x":-100,"y":0}}`,
	} {
		s := &Store{raw: map[string]json.RawMessage{"mygoDesktopPet": json.RawMessage(raw)}}
		got := s.DesktopPet()
		if got.Enabled || got.SkinID != "test" || !got.Neural || got.Behavior.Opacity != 1 || got.Position == nil || *got.Position != (PetPosition{X: -100, Y: 0}) {
			t.Fatalf("independent decode failed: %+v", got)
		}
	}
	for _, raw := range []string{`{}`, `{"behavior":null}`, `{"behavior":[]}`, `{"behavior":"invalid"}`} {
		s := &Store{raw: map[string]json.RawMessage{"mygoDesktopPet": json.RawMessage(raw)}}
		if got := s.DesktopPet(); got.Behavior != DefaultPetBehavior() {
			t.Fatalf("behavior defaults lost for %s: %+v", raw, got)
		}
	}
}

func TestDesktopPetRejectsMalformedPosition(t *testing.T) {
	for _, position := range []string{`null`, `[]`, `{}`, `{"x":1}`, `{"x":"1","y":2}`, `{"x":1,"y":null}`, `{"x":1.5,"y":2}`} {
		raw := `{"enabled":true,"position":` + position + `}`
		s := &Store{raw: map[string]json.RawMessage{"mygoDesktopPet": json.RawMessage(raw)}}
		got := s.DesktopPet()
		if got.Position != nil || !got.Enabled {
			t.Fatalf("position %s must be ignored independently: %+v", position, got)
		}
	}
}

func TestSaveDesktopPetBehaviorPreservesOtherConfig(t *testing.T) {
	dir := t.TempDir()
	initial := `{"port":4000,"pet":{"enabled":true,"xRatio":0.9},"live2dPet":{"enabled":true,"growth":{"points":7}},"pluginRecovery":{"reason":"manual"},"unknown":{"keep":[1,2,3]}}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := DesktopPet{Enabled: true, SkinID: "test", Neural: true, Behavior: DefaultPetBehavior(), Position: &PetPosition{X: -500, Y: 200}}
	want.Behavior.Scale, want.Behavior.ChatEnabled, want.Behavior.SelfTalk = 1.34, false, false
	if err := s.SaveDesktopPet(want); err != nil {
		t.Fatal(err)
	}
	want.Behavior.Scale = 1.3
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.DesktopPet(); !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]any
	if err := json.Unmarshal([]byte(initial), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &after); err != nil {
		t.Fatal(err)
	}
	delete(after, "mygoDesktopPet")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("unrelated config changed: %v -> %v", before, after)
	}
}

func TestSaveLegacyDesktopPetGetsBehaviorDefaults(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDesktopPet(DesktopPet{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	got := s.DesktopPet()
	if got.Behavior != DefaultPetBehavior() || got.Position != nil {
		t.Fatalf("legacy caller defaults = %+v", got)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(s.raw["mygoDesktopPet"], &raw); err != nil {
		t.Fatal(err)
	}
	if _, exists := raw["position"]; exists {
		t.Fatal("nil position must be omitted")
	}
	var behavior map[string]json.RawMessage
	if err := json.Unmarshal(raw["behavior"], &behavior); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"scale", "opacity", "personality", "activity", "selfTalk", "wander", "chatEnabled", "lockPosition", "shiftToDrag", "powerSave", "clickSound"} {
		if _, exists := behavior[field]; !exists {
			t.Fatalf("lowerCamel behavior field missing: %s", field)
		}
	}
}
