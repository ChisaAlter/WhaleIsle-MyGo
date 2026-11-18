package importer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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
	plain, _ := sessionLogFiles(dir)
	meta := readPlainSessionLog(context.Background(), plain)
	if meta.Title != "Hello World" || meta.Cwd != `C:\proj` {
		t.Fatalf("meta=%+v", meta)
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

// A profile's compressed logs are decoded by one process, not by one process
// per session: five hundred sessions must not mean five hundred spawns.
func TestSessionScanDecodesCompressedLogsInOneProcess(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	const sessions = 25
	for i := 0; i < sessions; i++ {
		writeFileAll(t, filepath.Join(src, "sessions", "ws", fmt.Sprintf("session-%02d", i), "session.v4.jsonl.zstd"), "packed")
	}

	var calls atomic.Int32
	previous := readCompressedSessionMeta
	readCompressedSessionMeta = func(_ context.Context, paths []string) map[string]sessionMeta {
		calls.Add(1)
		metas := make(map[string]sessionMeta, len(paths))
		for _, path := range paths {
			metas[path] = sessionMeta{Title: "title " + filepath.Base(filepath.Dir(path)), Cwd: "/ws"}
		}
		return metas
	}
	t.Cleanup(func() { readCompressedSessionMeta = previous })

	cat, err := BuildCatalog(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("%d compressed sessions opened %d decoders, want one", sessions, got)
	}
	if len(cat.Sessions) != sessions {
		t.Fatalf("listed %d sessions, want %d", len(cat.Sessions), sessions)
	}
	for _, row := range cat.Sessions {
		if !strings.HasPrefix(row.Label, "title session-") {
			t.Fatalf("row label %q did not come from the decoder", row.Label)
		}
	}
}
