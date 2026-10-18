package market

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	desktopmarket "whaleisle/internal/desktop/market"
)

func TestPersonalPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "market.json")
	s := NewWithPersonalPath(path)
	s.discovery.catalog.Items = []desktopmarket.CatalogItem{{ID: "owner/plugin"}, {ID: "other/plugin"}}
	if err := s.ToggleFavorite("owner/plugin"); err != nil {
		t.Fatal(err)
	}
	s.SetTab("favorites")
	if d := s.Discovery(); d.Total != 1 || d.Rows[0].ID != "owner/plugin" {
		t.Fatal(d)
	}
	if err := s.RecordInstall("owner/plugin", "applied", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordInstall("https://secret:password@example.com", "failed", "token=secret"); err != nil {
		t.Fatal(err)
	}
	s = NewWithPersonalPath(path)
	p := s.Personal()
	if !p.Favorites["owner/plugin"] || len(p.History) != 2 || p.History[1].Result != "applied" || time.Since(p.History[0].At) > time.Minute {
		t.Fatal(p)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "password") {
		t.Fatal(string(raw))
	}
	s.discovery.catalog.Items = []desktopmarket.CatalogItem{{ID: "owner/plugin"}}
	if err := s.ToggleFavorite("owner/plugin"); err != nil {
		t.Fatal(err)
	}
	if NewWithPersonalPath(path).Personal().Favorites["owner/plugin"] {
		t.Fatal("favorite removal lost")
	}
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	bad := NewWithPersonalPath(filepath.Join(blocker, "market.json"))
	bad.discovery.catalog.Items = s.discovery.catalog.Items
	if bad.ToggleFavorite("owner/plugin") == nil || bad.Personal().Err == "" || bad.Personal().Favorites["owner/plugin"] {
		t.Fatal("write failure not surfaced/rolled back")
	}
}
