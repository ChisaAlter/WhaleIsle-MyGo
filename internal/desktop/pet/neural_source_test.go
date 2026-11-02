package pet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"whaleisle/internal/config"
)

func TestRenderAssetSourceSelection(t *testing.T) {
	base := t.TempDir()
	exe := filepath.Join(base, "application", "whaleisle.exe")
	explicit := filepath.Join(base, "explicit", "pet-render-assets")
	adjacent := filepath.Join(filepath.Dir(exe), "pet-render-assets")
	for _, tc := range []struct {
		name, root, exe, want string
		development           bool
	}{
		{name: "release ignores absolute override", root: explicit, exe: exe, want: adjacent},
		{name: "release ignores invalid override", root: ".", exe: exe, want: adjacent},
		{name: "development default remains adjacent", development: true, exe: exe, want: adjacent},
		{name: "development absolute override", development: true, root: explicit, exe: exe, want: explicit},
		{name: "development cleans absolute override", development: true, root: explicit + string(filepath.Separator) + ".", exe: exe, want: explicit},
		{name: "relative override fails closed", development: true, root: "runtime/pet-render-assets", exe: exe},
		{name: "working directory is never selected", development: true, root: ".", exe: exe},
		{name: "missing executable has no adjacent source"},
		{name: "explicit source works without executable", development: true, root: explicit, want: explicit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectRenderRoot(tc.development, tc.root, tc.exe); got != tc.want {
				t.Fatalf("selected %q, want %q", got, tc.want)
			}
		})
	}
	if _, err := loadRenderAssets(""); err == nil {
		t.Fatal("empty source must not open the working directory")
	}
}

func TestConfiguredRenderAssetSourceFeedsRuntimeLookup(t *testing.T) {
	previous := configuredRenderRoot.Load()
	t.Cleanup(func() { configuredRenderRoot.Store(previous) })
	root := filepath.Join(t.TempDir(), "verified-assets")
	if got := ConfigureRenderAssets(true, root); got != root || petRenderRoot() != root {
		t.Fatalf("startup source not used by the runtime: %q / %q", got, petRenderRoot())
	}
	if got := ConfigureRenderAssets(true, "."); got != "" || petRenderRoot() != "" {
		t.Fatalf("invalid explicit source fell back: %q / %q", got, petRenderRoot())
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(filepath.Dir(exe), "pet-render-assets")
	if got := ConfigureRenderAssets(false, root); got != want || petRenderRoot() != want {
		t.Fatalf("packaged build accepted development override: %q / %q", got, petRenderRoot())
	}
}

func TestFreshAndSavedNeuralPreferenceSurvivesTrayToggle(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		want      bool
	}{
		{name: "fresh absent namespace", want: true},
		{name: "fresh empty document", raw: `{}`, want: true},
		{name: "existing explicit false", raw: `{"mygoDesktopPet":{"neural":false}}`},
		{name: "existing omitted false", raw: `{"mygoDesktopPet":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.raw != "" {
				if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(tc.raw), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			store, err := config.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			if store.DesktopPet().Neural != tc.want {
				t.Fatal("fixture renderer preference already changed")
			}
			// Keep readiness false: exercise the actual tray persistence path,
			// never create a window or discover skins in the user's home.
			p := &Companion{Store: store, Neural: !tc.want, SkinID: "unsubmitted-draft"}
			for _, enabled := range []bool{true, false, true} {
				if err := p.SetEnabled(enabled); err != nil {
					t.Fatal(err)
				}
				reopened, err := config.Open(dir)
				if err != nil {
					t.Fatal(err)
				}
				if got := reopened.DesktopPet(); got.Enabled != enabled || got.Neural != tc.want || got.SkinID != "" {
					t.Fatalf("tray toggle saved another draft or lost renderer preference: %+v", got)
				}
			}
		})
	}
}

func TestStagedRenderAssetSourceRejectsCorruptionWithoutAlternateSource(t *testing.T) {
	source := os.Getenv("WHALEISLE_PET_RENDER_ASSETS")
	if source == "" {
		t.Skip("WHALEISLE_PET_RENDER_ASSETS not set")
	}
	if !filepath.IsAbs(source) {
		t.Fatal("fixture source must be explicit and absolute")
	}
	if _, err := loadRenderAssets(source); err != nil {
		t.Fatalf("actual staged assets rejected before corruption fixture: %v", err)
	}
	manifest, err := assets.ReadFile("assets/render-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var list struct{ Files []renderAsset }
	if err := json.Unmarshal(manifest, &list); err != nil {
		t.Fatal(err)
	}
	copyRoot := t.TempDir()
	for _, file := range list.Files {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(file.Path)))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(copyRoot, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A sidecar cannot add scripts to the compiled route list.
	if err := os.WriteFile(filepath.Join(copyRoot, "provenance.json"), []byte(`{"files":[{"path":"extra.js"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(copyRoot, "extra.js"), []byte("throw Error('untrusted')"), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := configuredRenderRoot.Load()
	t.Cleanup(func() { configuredRenderRoot.Store(previous) })
	ConfigureRenderAssets(true, copyRoot)
	loaded, err := loadRenderAssets(petRenderRoot())
	if err != nil || len(loaded) != 7 || loaded["/render/extra.js"] != nil {
		t.Fatalf("compiled allowlist not enforced: %d routes, %v", len(loaded), err)
	}
	character := filepath.Join(copyRoot, "avatar", "character.png")
	data, err := os.ReadFile(character)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)/2] ^= 1
	if err := os.WriteFile(character, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err = loadRenderAssets(petRenderRoot())
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") || loaded != nil || petRenderRoot() != copyRoot {
		t.Fatalf("same-size corruption was admitted or switched source: %d routes, %v, root=%q", len(loaded), err, petRenderRoot())
	}
	ConfigureRenderAssets(true, filepath.Join(copyRoot, "missing"))
	if loaded, err := loadRenderAssets(petRenderRoot()); err == nil || loaded != nil {
		t.Fatal("missing explicit source admitted bytes from another directory")
	}
}
