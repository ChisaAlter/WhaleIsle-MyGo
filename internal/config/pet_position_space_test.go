package config

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDesktopPetPositionCoordinateSpace(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want *PetPosition
	}{
		{"legacy", `{"x":-500,"y":40}`, &PetPosition{X: -500, Y: 40}},
		{"physical", `{"x":-1000,"y":80,"space":"windows-physical"}`, &PetPosition{X: -1000, Y: 80, Space: PetPositionWindowsPhysical}},
		{"unknown", `{"x":1,"y":2,"space":"other"}`, nil},
		{"null", `{"x":1,"y":2,"space":null}`, nil},
		{"wrong type", `{"x":1,"y":2,"space":42}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &Store{raw: map[string]json.RawMessage{"mygoDesktopPet": json.RawMessage(`{"enabled":true,"position":` + tc.raw + `}`)}}
			got := store.DesktopPet()
			if !got.Enabled || !reflect.DeepEqual(got.Position, tc.want) {
				t.Fatalf("coordinate metadata decoded incorrectly: %#v", got)
			}
		})
	}
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := &PetPosition{X: -1000, Y: 80, Space: PetPositionWindowsPhysical}
	if err := store.SaveDesktopPet(DesktopPet{Enabled: true, Behavior: DefaultPetBehavior(), Position: want}); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reopened.DesktopPet().Position, want) {
		t.Fatal("physical coordinate marker was not persisted")
	}
}
