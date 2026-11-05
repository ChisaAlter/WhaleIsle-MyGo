package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, root, rel, body string) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRescanRetainsChoicesAndDistinctExtraSources(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	for _, name := range []string{"one", "two"} {
		put(t, source, "sessions/"+name+"/session.v3.jsonl", "{}\n")
	}
	a, b := t.TempDir(), t.TempDir()
	put(t, a, "same/SKILL.md", "first source")
	put(t, b, "same/SKILL.md", "second source")
	cat, err := BuildCatalog(source, dest)
	if err != nil {
		t.Fatal(err)
	}
	AppendSkillDirectory(&cat, a)
	AppendSkillDirectory(&cat, b)
	if len(cat.ExtraSkillRoots) != 2 {
		t.Fatal("extra source inventory lost")
	}
	var extra []CatalogItem
	for _, row := range cat.Skills {
		if strings.HasPrefix(row.ID, "extra") {
			extra = append(extra, row)
		}
	}
	if len(extra) != 2 || extra[0].ID == extra[1].ID {
		t.Fatalf("same-name roots collapsed: %+v", extra)
	}
	cat.Sessions[0].Selected = false
	cat.Skills[len(cat.Skills)-1].Selected = false
	put(t, source, "sessions/new/session.v3.jsonl", "{}\n")
	got, err := RescanCatalog(cat, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceHome != source || len(got.ExtraSkillRoots) != 2 {
		t.Fatal("rescan reset source/extra roots")
	}
	choices := map[string]bool{}
	for _, r := range got.Sessions {
		choices[r.Rel] = r.Selected
	}
	if choices[cat.Sessions[0].Rel] || !choices[cat.Sessions[1].Rel] || choices["new"] {
		t.Fatalf("rescan choices=%v", choices)
	}
	if got.Skills[len(got.Skills)-1].Selected {
		t.Fatal("extra skill choice reset")
	}
	offline := a + "-offline"
	if err := os.Rename(a, offline); err != nil {
		t.Fatal(err)
	}
	defer os.Rename(offline, a)
	missing, err := RescanCatalog(got, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(missing.ExtraSkillRoots) != 2 || missing.ExtraSkillRoots[0] != a {
		t.Fatal("temporarily unavailable extra root was forgotten")
	}
	for _, row := range missing.Skills {
		if row.Abs == extra[1].Abs && row.ID != extra[1].ID {
			t.Fatal("missing first root reassigned the second root identity")
		}
	}
}

func TestCatalogRejectsSQLiteOnlyAndMixedSessionLogs(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	put(t, source, "sessions/sqlite/session.db", "inert database")
	put(t, source, "sessions/mixed/session.v3.jsonl", "{}\n")
	put(t, source, "sessions/mixed/session.v3.jsonl.zstd", "inert compressed marker")
	cat, err := BuildCatalog(source, dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Sessions) != 2 {
		t.Fatalf("unsupported directories omitted: %+v", cat.Sessions)
	}
	for _, row := range cat.Sessions {
		if !row.Unsupported || row.Selected || row.UnsupportedReason == "" {
			t.Fatalf("unsafe session selectable: %+v", row)
		}
	}
	put(t, source, "sessions/ordinary/session.v3.jsonl", "{}\n")
	cat, err = BuildCatalog(source, dest)
	if err != nil {
		t.Fatal(err)
	}
	put(t, source, "sessions/ordinary/session.v3.jsonl.zstd", "encoding changed after scan")
	_, err = ApplyCatalog(cat, ApplySelection{SelectedRels: []string{"ordinary"}})
	if err == nil {
		t.Fatal("mixed encoding introduced after scan was copied")
	}
	if _, err = os.Stat(filepath.Join(dest, "sessions", "ordinary")); !os.IsNotExist(err) {
		t.Fatal("unsafe session was published")
	}
}

func TestSessionMetadataUsesLatestTitleBeyondScannerLimit(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "session.v3.jsonl", `{"type":"session","cwd":"C:\\fixture","title":"first"}`+"\n"+`{"type":"message","text":"`+strings.Repeat("x", 80000)+`"}`+"\n"+`{"type":"session/title","data":{"title":"latest title"}}`+"\n")
	title, cwd := readPlainSessionMeta(dir)
	if title != "latest title" || cwd != `C:\fixture` {
		t.Fatalf("metadata %q %q", title, cwd)
	}
}

func TestCompressedSessionMetadataUsesAllFrames(t *testing.T) {
	dir := t.TempDir()
	frame := func(s string) []byte {
		n := len(s)
		h := uint32(n<<3 | 1)
		return append([]byte{0x28, 0xb5, 0x2f, 0xfd, 0x20, byte(n), byte(h), byte(h >> 8), byte(h >> 16)}, []byte(s)...)
	}
	data := append(frame(`{"type":"session","cwd":"/fixture"}`+"\n"), frame(`{"type":"session/title","data":{"title":"compressed latest"}}`+"\n")...)
	if err := os.WriteFile(filepath.Join(dir, "session.v3.jsonl.zstd"), data, 0600); err != nil {
		t.Fatal(err)
	}
	title, cwd := readPlainSessionMeta(dir)
	if title != "compressed latest" || cwd != "/fixture" {
		t.Fatalf("compressed metadata %q %q (requires the existing Host Node zstd decoder)", title, cwd)
	}
}
