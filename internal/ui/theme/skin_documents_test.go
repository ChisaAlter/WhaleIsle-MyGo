package theme

import (
	"strings"
	"testing"
)

func TestSkinCopyExportImportRetainsExplicitPresentation(t *testing.T) {
	p := Defaults()
	for _, id := range []string{"codex", "claude", "deepseek"} {
		source := p.Family(id)
		copy := p.DuplicateFamily(source)
		if copy.Skin != source.Skin || copy.ID == source.ID {
			t.Fatal(id, copy)
		}
		raw := SerializeFamily(copy)
		if source.Skin == "" && strings.Contains(raw, `"skin"`) {
			t.Fatal("legacy default emitted skin metadata", raw)
		}
		got, err := p.ImportFamily(raw)
		if err != nil {
			t.Fatal(err)
		}
		if FamilySkin(got) != FamilySkin(source) {
			t.Fatal("skin lost across copy/import", id, got)
		}
		p.Custom = append(p.Custom, got)
		p.Light, p.Dark = got.ID, got.ID
		if SkinForMode(p, false) != FamilySkin(source) || SkinForMode(p, true) != FamilySkin(source) {
			t.Fatal("copied family did not own its skin", id)
		}
		p.Light, p.Dark = "deepseek", "deepseek"
	}
	unknown := p.DuplicateFamily(p.Family("deepseek"))
	unknown.Skin = "unsupported"
	p.Custom = append(p.Custom, unknown)
	if p.Validate() == nil {
		t.Fatal("unknown skin accepted")
	}
}
