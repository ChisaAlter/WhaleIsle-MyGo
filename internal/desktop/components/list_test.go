package components

import (
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/store/storelocale"
)

func TestListUserDataWhalebridgeCatalog(t *testing.T) {
	dir := t.TempDir()
	wb := filepath.Join(dir, "components", "whalebridge")
	if err := os.MkdirAll(wb, 0o755); err != nil {
		t.Fatal(err)
	}
	catalog := `{
  "manifest": {
    "id": "whalebridge",
    "name": "鲸桥",
    "version": "1.1.0",
    "description": "desc"
  }
}`
	if err := os.WriteFile(filepath.Join(wb, "catalog.json"), []byte(catalog), 0o644); err != nil {
		t.Fatal(err)
	}
	rows := ListUserData(dir)
	if len(rows) != 1 {
		t.Fatalf("rows=%d want 1", len(rows))
	}
	if rows[0].Name != storelocale.T(storelocale.KeyStoreComponentsWhalebridge) || rows[0].State != "available" || rows[0].Version != "1.1.0" {
		t.Fatalf("row=%+v", rows[0])
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.1.0", "1.0.6", 1},
		{"1.2.3-beta.1", "1.2.3", -1},
		{"1.2.3-rc.1", "1.2.3-beta.5", 1},
		{"1.2.3-beta.2", "1.2.3-beta.10", -1},
		{"v1.2.0", "1.2.0", 0},
		{"1.10.0", "1.9.0", 1},
		{"1.2.3+build.7", "1.2.3", 0},
	}
	for _, tc := range cases {
		if got := compareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestGenericDiskRowsPrereleaseOrdering(t *testing.T) {
	root := t.TempDir()
	samples := filepath.Join(root, "samples")
	writeSampleComponent(t, samples, "notes", "1.2.3")
	writeSampleComponent(t, samples, "notes", "1.2.3-beta.1")
	if err := os.MkdirAll(filepath.Join(root, "components"), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := `{"components":{"notes":{"version":"1.2.3-beta.1","state":"stopped","source":"bundled"}}}`
	if err := os.WriteFile(filepath.Join(root, "components", "registry.json"), []byte(registry), 0o600); err != nil {
		t.Fatal(err)
	}
	rows := genericDiskRows(root, samples)
	if len(rows) != 1 {
		t.Fatalf("rows = %#v", rows)
	}
	if rows[0].Version != "1.2.3" {
		t.Fatalf("catalog latest = %q, want 1.2.3", rows[0].Version)
	}
	if !rows[0].UpdateAvailable {
		t.Fatalf("installed 1.2.3-beta.1 with catalog 1.2.3 must offer the update: %#v", rows[0])
	}
}

func TestScanCatalogPrereleaseSortsBelowRelease(t *testing.T) {
	root := t.TempDir()
	samples := filepath.Join(root, "samples")
	writeSampleComponent(t, samples, "notes", "1.2.3")
	writeSampleComponent(t, samples, "notes", "1.2.3-beta.1")
	svc := newGenericService(filepath.Join(root, "user"), GenericOptions{SamplesRoot: samples}, fastGenericDeps())
	catalog := svc.scanCatalog()
	if len(catalog) != 1 {
		t.Fatalf("catalog = %#v", catalog)
	}
	if catalog[0].Latest != "1.2.3" {
		t.Fatalf("latest = %q, want 1.2.3", catalog[0].Latest)
	}
}
