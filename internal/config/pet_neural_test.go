package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDesktopPetNeuralDefaultPreservesExistingIntent(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		present   bool
		want      bool
	}{
		{name: "new namespace", want: true},
		{name: "old omitted false", raw: `{}`, present: true},
		{name: "old enabled omitted false", raw: `{"enabled":true}`, present: true},
		{name: "explicit false", raw: `{"neural":false}`, present: true},
		{name: "explicit true", raw: `{"neural":true}`, present: true, want: true},
		{name: "null field", raw: `{"neural":null}`, present: true},
		{name: "string field", raw: `{"neural":"true"}`, present: true},
		{name: "number field", raw: `{"neural":1}`, present: true},
		{name: "null namespace", raw: `null`, present: true},
		{name: "array namespace", raw: `[]`, present: true},
		{name: "invalid namespace", raw: `false`, present: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]json.RawMessage{"live2dPet": json.RawMessage(`{"enabled":true}`)}
			if tc.present {
				raw["mygoDesktopPet"] = json.RawMessage(tc.raw)
			}
			store := &Store{raw: raw}
			got := store.DesktopPet()
			if got.Neural != tc.want || got.Behavior != DefaultPetBehavior() {
				t.Fatalf("neural=%v behavior=%+v, want neural=%v and default behavior", got.Neural, got.Behavior, tc.want)
			}
			if tc.name != "old enabled omitted false" && got.Enabled {
				t.Fatal("renderer preference enabled the companion window")
			}
			if _, present := store.raw["mygoDesktopPet"]; present != tc.present {
				t.Fatal("read materialized a namespace or changed original configuration")
			}
		})
	}
}

func TestDesktopPetNeuralFreshNamespaceSurvivesGeneralSave(t *testing.T) {
	for _, initial := range []string{"", `{}`} {
		dir := t.TempDir()
		if initial != "" {
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(initial), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		store, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := store.raw["mygoDesktopPet"]; exists {
			t.Fatal("fixture unexpectedly contains a companion namespace")
		}
		if err := store.Update(func(cfg *Config) { cfg.Port = 4001 }); err != nil {
			t.Fatal(err)
		}
		reopened, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		value := reopened.DesktopPet()
		if !value.Neural || value.Enabled {
			t.Fatalf("general save changed fresh defaults: %+v", value)
		}
		value.Behavior.Opacity = 0.7
		if err := reopened.SaveDesktopPet(value); err != nil {
			t.Fatal(err)
		}
		reopened, err = Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got := reopened.DesktopPet(); !got.Neural || got.Enabled || got.Behavior.Opacity != 0.7 {
			t.Fatalf("first pet appearance save lost default renderer: %+v", got)
		}
	}
}

func TestDesktopPetNeuralExplicitFalseSurvivesSaveReopenAndOtherUpdates(t *testing.T) {
	dir := t.TempDir()
	initial := `{"port":4000,"pet":{"enabled":true,"petId":"original"},"live2dPet":{"enabled":true,"x":25},"unknown":{"keep":[1,2,3]},"mygoDesktopPet":{"enabled":false,"skinId":"saved-skin","neural":false,"position":{"x":-200,"y":75,"space":"windows-physical"},"behavior":{"scale":1.3,"opacity":0.7,"wander":false}}}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := store.DesktopPet()
	if want.Neural || want.Enabled {
		t.Fatal("saved disabled state changed")
	}
	for _, enabled := range []bool{false, true, false} {
		want.Neural = enabled
		if err := store.SaveDesktopPet(want); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(store.Path())
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]json.RawMessage
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		var pet map[string]json.RawMessage
		if err := json.Unmarshal(document["mygoDesktopPet"], &pet); err != nil {
			t.Fatal(err)
		}
		if string(pet["neural"]) != map[bool]string{true: "true", false: "false"}[enabled] {
			t.Fatalf("explicit neural=%v not persisted: %s", enabled, document["mygoDesktopPet"])
		}
		store, err = Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got := store.DesktopPet(); !reflect.DeepEqual(got, want) {
			t.Fatalf("round trip changed companion: %+v -> %+v", want, got)
		}
	}
	if err := store.Update(func(cfg *Config) { cfg.Port = 4001 }); err != nil {
		t.Fatal(err)
	}
	want.Behavior.Opacity = 0.6
	if err := store.SaveDesktopPet(want); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.DesktopPet(); !reflect.DeepEqual(got, want) || got.Neural || got.Enabled {
		t.Fatalf("unrelated updates changed saved state: %+v", got)
	}
	var before map[string]json.RawMessage
	if err := json.Unmarshal([]byte(initial), &before); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pet", "live2dPet", "unknown"} {
		var a, b any
		_ = json.Unmarshal(before[name], &a)
		_ = json.Unmarshal(reopened.raw[name], &b)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("unrelated %s changed", name)
		}
	}
}

func TestDesktopPetNeuralLegacyMissingRemainsOffAfterSave(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"mygoDesktopPet":{"enabled":true,"skinId":"legacy"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	value := store.DesktopPet()
	if value.Neural || !value.Enabled || value.SkinID != "legacy" {
		t.Fatalf("legacy preference changed: %+v", value)
	}
	value.Behavior.Scale = 1.2
	if err := store.SaveDesktopPet(value); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.DesktopPet(); got.Neural || !got.Enabled || got.Behavior.Scale != 1.2 || got.SkinID != "legacy" {
		t.Fatalf("legacy preference changed after unrelated save: %+v", got)
	}
}
