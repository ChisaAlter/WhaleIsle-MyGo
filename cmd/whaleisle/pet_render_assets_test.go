package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPetRenderAssetsStartupHonorsDevelopmentBoundary(t *testing.T) {
	previous := buildVersion
	t.Cleanup(func() {
		buildVersion = previous
		configurePetRenderAssets()
	})
	explicit := filepath.Join(t.TempDir(), "pet-render-assets")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	adjacent := filepath.Join(filepath.Dir(exe), "pet-render-assets")
	for _, tc := range []struct {
		name, version, input, want string
	}{
		{"go run default", "dev", "", adjacent},
		{"go run explicit source", "dev", explicit, explicit},
		{"local development package", "0.0.0-dev", explicit, explicit},
		{"invalid development path", "dev", "runtime/pet-render-assets", ""},
		{"release ignores explicit source", "1.4.2", explicit, adjacent},
		{"release ignores invalid source", "1.4.2", ".", adjacent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buildVersion = tc.version
			t.Setenv("WHALEISLE_PET_RENDER_ASSETS", tc.input)
			if got := configurePetRenderAssets(); got != tc.want {
				t.Fatalf("version=%q source=%q resolved to %q, want %q", tc.version, tc.input, got, tc.want)
			}
		})
	}
}
