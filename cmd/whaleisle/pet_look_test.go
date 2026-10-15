package main

import (
	"testing"

	settingsstore "whaleisle/internal/store/settings"
)

func TestPetLookRequiresConfiguredImageRoute(t *testing.T) {
	valid := settingsstore.VisionView{Present: true, Writable: true, Provider: "p", Model: "image", Routes: []settingsstore.VisionRoute{{Provider: "p", Model: "image"}}}
	if p, m, ok := petLookRoute(valid); !ok || p != "p" || m != "image" {
		t.Fatal("configured route rejected")
	}
	for _, mutate := range []func(*settingsstore.VisionView){
		func(v *settingsstore.VisionView) { v.Present = false },
		func(v *settingsstore.VisionView) { v.Writable = false },
		func(v *settingsstore.VisionView) { v.Loading = true },
		func(v *settingsstore.VisionView) { v.Saving = true },
		func(v *settingsstore.VisionView) { v.Err = "offline" },
		func(v *settingsstore.VisionView) { v.Model = "text-only" },
		func(v *settingsstore.VisionView) { v.Provider = "other" },
		func(v *settingsstore.VisionView) { v.Routes = nil },
	} {
		v := valid
		mutate(&v)
		if _, _, ok := petLookRoute(v); ok {
			t.Fatalf("accepted unavailable route: %+v", v)
		}
	}
}
