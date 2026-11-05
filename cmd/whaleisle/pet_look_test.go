package main

import (
	"testing"
	"whaleisle/internal/config"
	"whaleisle/internal/petlook"
)

func TestPetLookRequiresConfiguredImageRoute(t *testing.T) {
	b := config.DefaultPetBehavior()
	b.LookProvider, b.LookModel = "pet-provider", "pet-image"
	cat := petlook.Catalog{Connected: true, Routes: []petlook.Route{{Provider: "pet-provider", Model: "pet-image"}, {Provider: "global-provider", Model: "global-image"}}}
	if p, m, ok := petLookRoute(b, cat); !ok || p != "pet-provider" || m != "pet-image" {
		t.Fatal("pet-specific pair rejected")
	}
	for _, mutate := range []func(*petlook.Catalog){
		func(c *petlook.Catalog) { c.Connected = false },
		func(c *petlook.Catalog) { c.Loading = true },
		func(c *petlook.Catalog) { c.Err = "offline" },
		func(c *petlook.Catalog) { c.Routes = nil },
	} {
		v := cat
		mutate(&v)
		if _, _, ok := petLookRoute(b, v); ok {
			t.Fatal("unavailable image route admitted")
		}
	}
	b.LookProvider = ""
	// The installed model-only desktop route remains available without a
	// catalog provider; it must not silently choose one from the catalog.
	if p, m, ok := petLookRoute(b, cat); !ok || p != "" || m != b.LookModel {
		t.Fatalf("explicit desktop route changed destination: provider=%q model=%q available=%v", p, m, ok)
	}
	b.LookModel = ""
	if _, _, ok := petLookRoute(b, cat); ok {
		t.Fatal("default Off fell back to another route")
	}
}
