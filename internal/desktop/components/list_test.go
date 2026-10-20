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
	if compareVersions("1.1.0", "1.0.6") <= 0 {
		t.Fatal("1.1.0 should be newer than 1.0.6")
	}
}
