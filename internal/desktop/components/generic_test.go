package components

import (
	"os"
	"path/filepath"
	"testing"
)

// No component process, installation, or network access: only private fixture files.
func TestGenericCatalogAndOriginalRegistry(t *testing.T) {
	root := t.TempDir()
	samples := filepath.Join(root, "samples")
	payload := filepath.Join(samples, "notes", "2.0.0")
	if err := os.MkdirAll(payload, 0755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"manifest.json": `{"id":"notes","name":"Notes","version":"2.0.0","entry":"notes.js","kind":"service"}`, "notes.js": ""} {
		if err := os.WriteFile(filepath.Join(payload, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "components"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "components", "registry.json"), []byte(`{"components":{"notes":{"version":"1.0.0","previous":"0.9.0","state":"stopped","source":"bundled"},"orphan":{"version":"1.0.0","state":"error","lastError":"missing payload"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	rows := genericDiskRows(root, samples)
	if len(rows) != 2 {
		t.Fatalf("rows = %#v", rows)
	}
	if rows[0].ID != "notes" || rows[0].PreviousVersion != "0.9.0" || !rows[0].UpdateAvailable || rows[0].Kind != "service" {
		t.Fatalf("catalog/registry mismatch: %#v", rows[0])
	}
	if rows[1].ID != "orphan" || !rows[1].Orphan || rows[1].Message != "missing payload" {
		t.Fatalf("orphan missing: %#v", rows[1])
	}
}

func TestGenericManifestEntryContainment(t *testing.T) {
	for _, entry := range []string{"../escape.js", `C:\escape.js`, "/escape.js", `..\escape.js`} {
		if genericEntry("/payload", entry) != "" {
			t.Fatalf("accepted escaping entry %q", entry)
		}
	}
	if genericEntry("/payload", "nested/entry.js") == "" {
		t.Fatal("rejected relative entry")
	}
}
