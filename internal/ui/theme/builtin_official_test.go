package theme

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func installedBuiltinFamilies(t *testing.T) []Family {
	t.Helper()
	data, err := os.ReadFile("testdata/installed-0.3.10-builtins.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		AppVersion     string   `json:"appVersion"`
		PackageVersion string   `json:"packageVersion"`
		SourceSHA256   string   `json:"sourceSHA256"`
		BundleSHA256   string   `json:"bundleSHA256"`
		Families       []Family `json:"families"`
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	if source.AppVersion != "0.3.10" || source.PackageVersion != "0.2.1-rc.2" || source.SourceSHA256 != "1d6a334d9b4e82ef0d258e6f3439fb56577dbe0622b44421ef01828757555cf6" || source.BundleSHA256 != "12a8aad0a672ac6f798e9247fbf87bc9434220360d453592599fcfe81822d623" {
		t.Fatal("builtin fixture does not identify the inspected installed build")
	}
	return source.Families
}

func TestBuiltinLibraryMatchesInstalledSource(t *testing.T) {
	source := installedBuiltinFamilies(t)
	got := Library()
	if len(got) != len(source) {
		t.Fatalf("builtin count=%d want installed %d", len(got), len(source))
	}
	for i, want := range source {
		family := got[i]
		if family.ID != want.ID || strings.SplitN(family.Name, " / ", 2)[0] != want.Name {
			t.Fatalf("builtin %d identity=%q/%q want %q/%q", i, family.ID, family.Name, want.ID, want.Name)
		}
		if FamilySkin(family) != FamilySkin(want) {
			t.Errorf("%s skin=%q want installed %q", family.ID, FamilySkin(family), FamilySkin(want))
		}
		if !reflect.DeepEqual(family.Light, want.Light) || !reflect.DeepEqual(family.Dark, want.Dark) {
			t.Errorf("%s seeds or overrides differ from installed source", family.ID)
		}
	}
}

func TestOfficialBuiltinPalettesResolveHostSelectedHalves(t *testing.T) {
	previous := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(previous) })
	roles := map[string]string{
		"BG": "--dsw-alias-bg-base", "FG": "--dsw-alias-label-primary",
		"Field": "--dsw-alias-bg-layer-1", "SelectorFill": "--dsw-specific-selector",
		"Titlebar": "--dsw-alias-bg-overlay", "InputMajor": "--dsw-specific-input-major",
		"Muted": "--dsw-alias-label-tertiary", "Secondary": "--dsw-alias-label-secondary", "Caption": "--dsw-alias-label-caption",
		"Accent": "--dsw-alias-state-business-primary", "ButtonFG": "--dsw-alias-label-primary-foreground",
		"BorderL1": "--dsw-alias-border-l1", "Border": "--dsw-alias-border-l2", "BorderL3": "--dsw-alias-border-l3", "BorderL4": "--dsw-alias-border-l4",
		"BrandPrimary": "--dsw-alias-button-primary-fill", "BrandPrimaryHover": "--dsw-alias-button-primary-hover",
		"ButtonInfoFill": "--dsw-alias-button-info-fill", "ButtonInfoHover": "--dsw-alias-button-info-hover",
		"Hover": "--dsw-alias-interactive-bg-hover", "Pressed": "--dsw-alias-interactive-bg-active",
		"Bubble": "--dsw-specific-bubble", "BubbleHighlight": "--dsw-specific-bubble-highlight", "Sidebar": "--dsw-specific-sidebar-fill",
		"CodeBlock": "--dsw-alias-markdown-code-block", "Danger": "--dsw-alias-state-error-primary",
	}
	for _, family := range installedBuiltinFamilies(t) {
		if family.ID != "codex" && family.ID != "claude" {
			continue
		}
		t.Run(family.ID, func(t *testing.T) {
			host, err := ReadHostSettings(json.RawMessage(`{"activeLightThemeId":"` + family.ID + `","activeDarkThemeId":"` + family.ID + `"}`))
			if err != nil {
				t.Fatal(err)
			}
			prefs := host.Preferences()
			if err := prefs.Validate(); err != nil {
				t.Fatal(err)
			}
			SetPreferences(prefs)
			for _, dark := range []bool{false, true} {
				seed, pinned := family.Light, IDDeepseekLight
				if dark {
					seed, pinned = family.Dark, IDDeepseekDark
				}
				wantScheme := SchemeLight
				if dark {
					wantScheme = SchemeDark
				}
				got := ResolveWithSystem(IDSystem, dark)
				if got.Scheme != wantScheme || Resolve(pinned).Scheme != wantScheme {
					t.Fatal("selecting a family changed the resolved scheme")
				}
				for _, colors := range []Colors{got.Active(dark), Resolve(pinned).Active(!dark)} {
					v := reflect.ValueOf(colors)
					for role, token := range roles {
						want, present := seed.Overrides[token]
						if !present {
							t.Fatalf("source missing tested semantic token %s", token)
						}
						if value := v.FieldByName(role).String(); value != want {
							t.Errorf("dark=%v %s=%q want installed %q", dark, role, value, want)
						}
					}
				}
			}
		})
	}
}

func TestOfficialBuiltinHalvesRemainIndependentAndCopiesOwnOverrides(t *testing.T) {
	previous := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(previous) })
	prefs := Defaults()
	prefs.Light, prefs.Dark = "codex", "claude"
	SetPreferences(prefs)
	resolved := Resolve(IDSystem)
	if resolved.Light.BG != "#ffffff" || resolved.Light.Accent != "#2563eb" || resolved.Dark.BG != "#262624" || resolved.Dark.Accent != "#d97757" {
		t.Fatal("independent Host halves were not resolved", resolved)
	}
	for _, id := range []string{"codex", "claude"} {
		source := prefs.Family(id)
		copy := prefs.DuplicateFamily(source)
		if copy.ID == id || !reflect.DeepEqual(copy.Light, source.Light) || !reflect.DeepEqual(copy.Dark, source.Dark) {
			t.Fatalf("duplicate lost source %s", id)
		}
		copy.Light.Overrides["--dsw-specific-input-major"] = "#010203"
		copy.Dark.Overrides["--dsw-specific-input-major"] = "#040506"
		if source.Light.Overrides["--dsw-specific-input-major"] == "#010203" || source.Dark.Overrides["--dsw-specific-input-major"] == "#040506" {
			t.Fatalf("custom copy mutated builtin %s", id)
		}
		prefs.Custom = append(prefs.Custom, source)
		if err := prefs.Validate(); err == nil {
			t.Fatalf("builtin id %s was not reserved", id)
		}
		prefs.Custom = nil
	}
}
