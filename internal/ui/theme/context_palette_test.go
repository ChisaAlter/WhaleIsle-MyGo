package theme

import "testing"

func TestContextPaletteCustomFamiliesAndOverrides(t *testing.T) {
	contrast := float64(50)
	for _, dark := range []bool{false, true} {
		seed := Seed{Background: "#ffffff", Foreground: "#102030", Accent: "#40668f", Contrast: &contrast}
		base := deepseekLight
		if dark {
			seed.Background, seed.Foreground = "#101820", "#eef4fa"
			base = deepseekDark
		}
		got := palette(seed, dark)
		if got.Secondary != mix(seed.Foreground, seed.Background, .31) || got.Muted != base.Muted || got.Caption != base.Caption {
			t.Fatalf("dark=%v custom labels=%q/%q/%q", dark, got.Secondary, got.Muted, got.Caption)
		}
		if got.BorderL1 != familyAlpha(seed.Foreground, .17) || got.Border == got.BorderL1 {
			t.Fatalf("custom border roles collapsed: %+v", got)
		}
		seed.Overrides = map[string]string{"--dsw-alias-border-l1": "#112233", "--dsw-alias-border-l2": "#223344", "--dsw-alias-markdown-code-block": "#334455", "--dsw-alias-interactive-bg-hover": "#445566", "--dsw-alias-interactive-bg-active": "#556677", "--dsw-alias-label-tertiary": "#667788", "--dsw-alias-label-caption": "#778899"}
		got = palette(seed, dark)
		if got.BorderL1 != "#112233" || got.Border != "#223344" || got.Muted != "#667788" || got.Caption != "#778899" {
			t.Fatal("explicit semantic overrides did not win", got)
		}
		wantBG, wantHover := "#334455", "#445566"
		if dark {
			wantBG, wantHover = "#445566", "#556677"
		}
		if got.TriggerBG != wantBG || got.TriggerHover != wantHover {
			t.Fatal("trigger aliases did not follow underlying overrides", got)
		}
		seed.Overrides["--dsw-alias-turn-trigger-bg"] = "#8899aa"
		seed.Overrides["--dsw-alias-turn-trigger-bg-hover"] = "#99aabb"
		got = palette(seed, dark)
		if got.TriggerBG != "#8899aa" || got.TriggerHover != "#99aabb" {
			t.Fatal("trigger overrides lost", got)
		}
	}
}
