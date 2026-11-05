package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildCatalogListsSessionsSkillsSettings(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	sess := filepath.Join(src, "sessions", "ws", "abc")
	if err := os.MkdirAll(sess, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sess, "session.v3.jsonl"), "{}\n")
	skill := filepath.Join(src, "skills", "demo")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(skill, "SKILL.md"), "# demo\n")
	writeFile(t, filepath.Join(src, "settings.yaml"), "llm-deepseek:\n  model: x\nui-theme:\n  mode: light\n")
	writeFile(t, filepath.Join(src, "AGENTS.md"), "hi\n")
	preset := filepath.Join(src, ".agent-presets", "coder")
	if err := os.MkdirAll(preset, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(preset, "agent.cordis.yml"), "name: coder\n")

	cat, err := BuildCatalog(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Sessions) != 1 || cat.Sessions[0].Rel != "ws/abc" {
		t.Fatalf("sessions = %+v", cat.Sessions)
	}
	if !cat.Sessions[0].Selected {
		t.Fatal("session should default selected")
	}
	foundDemo := false
	for _, s := range cat.Skills {
		if s.ID == "home:demo" {
			foundDemo = true
			break
		}
	}
	if !foundDemo {
		t.Fatalf("skills missing home:demo: %+v", cat.Skills)
	}
	if len(cat.Settings) < 2 {
		t.Fatalf("settings = %+v", cat.Settings)
	}
	if len(cat.Presets) != 1 || cat.Presets[0].ID != "coder" {
		t.Fatalf("presets = %+v", cat.Presets)
	}
	counts := cat.CategoryCounts()
	if counts["sessions"][1] != 1 || counts["skills"][1] < 1 {
		t.Fatalf("counts = %+v", counts)
	}
}

func TestApplyCatalogCopiesSelected(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	sess := filepath.Join(src, "sessions", "a")
	if err := os.MkdirAll(sess, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sess, "session.jsonl"), "x\n")
	skill := filepath.Join(src, "skills", "s1")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(skill, "SKILL.md"), "# s1\n")

	cat, err := BuildCatalog(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	sel := SelectionFromCatalog(cat, false, true)
	res, err := ApplyCatalog(cat, sel)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Copied < 2 {
		t.Fatalf("res = %+v", res)
	}
	if !entryExists(filepath.Join(dst, "sessions", "a", "session.jsonl"), false) {
		t.Fatal("session not copied")
	}
	if !entryExists(filepath.Join(dst, "skills", "s1", "SKILL.md"), false) {
		t.Fatal("skill not copied")
	}
}

func TestReadPlainSessionMeta(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "session.v3.jsonl"),
		`{"type":"session","title":"Hello World","cwd":"C:\\proj"}`+"\n"+`{"type":"message"}`+"\n")
	title, cwd := readPlainSessionMeta(dir)
	if title != "Hello World" || cwd != `C:\proj` {
		t.Fatalf("title=%q cwd=%q", title, cwd)
	}
}

func TestApplyCatalogEmptySelection(t *testing.T) {
	cat := Catalog{SourceHome: t.TempDir(), DestHome: t.TempDir()}
	res, err := ApplyCatalog(cat, ApplySelection{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Empty || !res.OK {
		t.Fatalf("res = %+v", res)
	}
}

func TestAppendSkillDirectory(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "myskill")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("# s"), 0o644); err != nil {
		t.Fatal(err)
	}
	cat := Catalog{SourceHome: t.TempDir(), DestHome: t.TempDir()}
	n := AppendSkillDirectory(&cat, root)
	if n != 1 || len(cat.Skills) != 1 || cat.Skills[0].Label != "myskill" {
		t.Fatalf("n=%d skills=%+v", n, cat.Skills)
	}
	if AppendSkillDirectory(&cat, root) != 0 {
		t.Fatal("duplicate root should add 0")
	}
}
