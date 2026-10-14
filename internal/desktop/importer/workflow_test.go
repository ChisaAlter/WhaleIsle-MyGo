package importer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportWorkflowFixtures(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	source := filepath.Join(home, "source")
	dest := filepath.Join(home, "dest")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(p, value string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(source, "sessions", "a", "session.v3.jsonl"), "{}\n")
	write(filepath.Join(source, "sessions", "b", "session.v3.jsonl"), "{}\n")
	write(filepath.Join(source, "settings.yaml"), "ui-theme:\n  mode: dark\nother:\n  keep: source\n")
	cat, err := BuildCatalog(source, dest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("preview wrote destination")
	}
	clone := cat.Snapshot()
	clone.Sessions[0].Selected = false
	if !cat.Sessions[0].Selected {
		t.Fatal("snapshot shares checkbox state")
	}
	res, err := ApplyCatalogProgress(context.Background(), cat, ApplySelection{}, nil)
	if err != nil || !res.Empty {
		t.Fatalf("empty: %+v %v", res, err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("empty selection wrote destination")
	}
	ctx, cancel := context.WithCancel(context.Background())
	events := 0
	res, err = ApplyCatalogProgress(ctx, cat, ApplySelection{SelectedRels: []string{"a", "b"}}, func(p ApplyResult) { events++; cancel() })
	if err != context.Canceled || res.Copied != 1 || len(res.Journal) < 2 || events != 1 {
		t.Fatalf("partial cancellation: %+v %v events=%d", res, err, events)
	}
	if _, err := os.Stat(filepath.Join(dest, "sessions", "b")); !os.IsNotExist(err) {
		t.Fatal("copied after cancellation")
	}
	// A later failure retains earlier completed records.
	broken := cat.Snapshot()
	broken.Sessions[1].Abs = filepath.Join(source, "missing")
	res, err = ApplyCatalogProgress(context.Background(), broken, ApplySelection{SelectedRels: []string{"a", "b"}}, nil)
	if err == nil || res.Copied != 1 || len(res.Journal) < 2 {
		t.Fatalf("partial failure: %+v %v", res, err)
	}
	write(filepath.Join(dest, "settings.yaml"), "ui-theme:\n  mode: original\n")
	sel := ApplySelection{SelectedSettingIDs: []string{"ui-theme"}}
	res, err = ApplyCatalog(cat, sel)
	data, _ := os.ReadFile(filepath.Join(dest, "settings.yaml"))
	if err != nil || res.Skipped != 1 || string(data) != "ui-theme:\n  mode: original\n" {
		t.Fatalf("default conflict: %+v %v %q", res, err, data)
	}
	sel.Overwrite = true
	confirmation := Confirmation(cat, sel)
	for _, warning := range []string{source, dest, "只替换所选同名项", "引用凭据不迁移", "未选节"} {
		if !strings.Contains(confirmation, warning) {
			t.Fatalf("missing disclosure %q", warning)
		}
	}
	res, err = ApplyCatalog(cat, sel)
	data, _ = os.ReadFile(filepath.Join(dest, "settings.yaml"))
	if err != nil || res.Copied != 1 || strings.Contains(string(data), "other:") || !strings.Contains(string(data), "mode: dark") {
		t.Fatalf("explicit selected-section merge: %+v %v", res, err)
	}
	for _, pair := range [][2]string{{source, source}, {source, filepath.Join(source, "nested")}, {filepath.Join(source, "nested"), source}} {
		if ValidateRoots(pair[0], pair[1]) == nil {
			t.Fatalf("accepted overlapping roots %v", pair)
		}
	}
	original, _ := os.ReadFile(filepath.Join(source, "sessions", "a", "session.v3.jsonl"))
	if string(original) != "{}\n" {
		t.Fatal("source changed")
	}
}
