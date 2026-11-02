package theme

import "testing"

func TestSkinLayoutInstalledMetricsAndSelectedHalf(t *testing.T) {
	old := CurrentPreferences()
	t.Cleanup(func() { SetPreferences(old) })
	p := Defaults()
	p.Light, p.Dark = "codex", "claude"
	SetPreferences(p)
	l := ResolvedSkinLayout(IDSystem, false)
	if l.ContentWidth != 720 || l.ComposerRadius != 18 || l.ComposerBorder != .5 || l.SidebarInline != 10 || l.BubbleRadii != ([4]float32{18, 18, 4, 18}) || l.ComposerBackground != "#ffffff" {
		t.Fatalf("codex=%+v", l)
	}
	l = ResolvedSkinLayout(IDDeepseekDark, false)
	if l.Skin != SkinClaude || l.ComposerRadius != 22 || l.ComposerBorder != 1 || l.HeaderBorder != 1 || l.BubbleRadii != ([4]float32{12, 12, 12, 12}) || l.ComposerBackground != "#2c2c2a" {
		t.Fatalf("claude=%+v", l)
	}
	copy := p.Family("codex")
	copy.ID = "customskin"
	copy.Name = "Custom Codex"
	copy.Dark = copy.Dark.Clone()
	copy.Dark.Overrides["--dsw-alias-bg-layer-1"] = "#123456"
	p.Custom = append(p.Custom, copy)
	p.Dark = copy.ID
	SetPreferences(p)
	l = ResolvedSkinLayout(IDSystem, true)
	if l.Skin != SkinCodex || l.ComposerBackground != "#123456" {
		t.Fatalf("explicit custom skin=%+v", l)
	}
	p.Light, p.Dark = "deepseek", "deepseek"
	SetPreferences(p)
	l = ResolvedSkinLayout(IDSystem, false)
	if l.Active() || l.ContentWidth != 0 || l.ComposerRadius != 28 || l.SidebarInline != 12 || l.BubbleRadii[0] != 20 {
		t.Fatalf("default changed %+v", l)
	}
}
