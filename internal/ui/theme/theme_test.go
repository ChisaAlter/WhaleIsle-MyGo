package theme

import (
	"strings"
	"testing"
)

func TestResolveDefaultDeepseek(t *testing.T) {
	for _, id := range []string{"", "deepseek", "DeepSeek", "  deepseek  "} {
		got := Resolve(id)
		if got.ID != DefaultID {
			t.Fatalf("Resolve(%q).ID = %q, want %q", id, got.ID, DefaultID)
		}
		if got.Scheme != SchemeSystem {
			t.Fatalf("Resolve(%q).Scheme = %q, want %q", id, got.Scheme, SchemeSystem)
		}
		if got.Light.BG != "#ffffff" || got.Dark.BG != "#151517" {
			t.Fatalf("Resolve(%q) palette drift: light=%q dark=%q", id, got.Light.BG, got.Dark.BG)
		}
		if got.Light.Accent != "#4176e6" || got.Dark.Accent != "#679efe" {
			t.Fatalf("Resolve(%q) accent drift: light=%q dark=%q", id, got.Light.Accent, got.Dark.Accent)
		}
	}
}

func TestResolveDarkAndLightVariants(t *testing.T) {
	dark := Resolve(IDDeepseekDark)
	if dark.Scheme != SchemeDark || dark.BackgroundColor() != "#151517" {
		t.Fatalf("dark variant: scheme=%q bg=%q", dark.Scheme, dark.BackgroundColor())
	}
	if dark.Active(false).BG != "#151517" {
		t.Fatalf("dark Active ignored scheme: %q", dark.Active(false).BG)
	}

	light := Resolve("light")
	if light.Scheme != SchemeLight || light.BackgroundColor() != "#ffffff" {
		t.Fatalf("light variant: scheme=%q bg=%q", light.Scheme, light.BackgroundColor())
	}
}

func TestResolveSystemAndUnknown(t *testing.T) {
	sys := Resolve(IDSystem)
	if sys.ID != DefaultID || sys.Scheme != SchemeSystem {
		t.Fatalf("system: %+v", sys)
	}
	unknown := Resolve("not-a-theme")
	if unknown.ID != DefaultID || unknown.Scheme != SchemeSystem {
		t.Fatalf("unknown should fall back to deepseek/system: %+v", unknown)
	}
}

func TestBackgroundColorLightDark(t *testing.T) {
	bg := Default().BackgroundColor()
	want := "light-dark(#ffffff, #151517)"
	if bg != want {
		t.Fatalf("BackgroundColor() = %q, want %q", bg, want)
	}
	if !strings.HasPrefix(bg, "light-dark(") || !strings.HasSuffix(bg, ")") {
		t.Fatalf("BackgroundColor() not a light-dark() form: %q", bg)
	}
}

func TestResolveWithSystem(t *testing.T) {
	light := ResolveWithSystem(DefaultID, false)
	if light.Scheme != SchemeLight || light.BackgroundColor() != "#ffffff" {
		t.Fatalf("system→light: scheme=%q bg=%q", light.Scheme, light.BackgroundColor())
	}
	dark := ResolveWithSystem(IDSystem, true)
	if dark.Scheme != SchemeDark || dark.BackgroundColor() != "#151517" {
		t.Fatalf("system→dark: scheme=%q bg=%q", dark.Scheme, dark.BackgroundColor())
	}
	pinned := ResolveWithSystem(IDDeepseekDark, false)
	if pinned.Scheme != SchemeDark {
		t.Fatalf("pinned dark should ignore systemDark: %q", pinned.Scheme)
	}
}

func TestSidebarAndTitlebarPresent(t *testing.T) {
	th := Resolve(DefaultID)
	if th.Light.Sidebar == "" || th.Dark.Sidebar == "" {
		t.Fatal("sidebar colors missing")
	}
	if th.Light.Titlebar == "" || th.Dark.Titlebar == "" {
		t.Fatal("titlebar colors missing")
	}
	if th.Light.Border == "" || th.Dark.Border == "" {
		t.Fatal("border colors missing")
	}
}
