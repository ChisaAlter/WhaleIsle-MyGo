package market

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
)

type hangingCatalog struct{}

func (hangingCatalog) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{}, nil
}
func (hangingCatalog) Details(ctx context.Context, _ desktopmarket.CatalogItem, _ bool) (desktopmarket.Details, error) {
	<-ctx.Done()
	return desktopmarket.Details{}, ctx.Err()
}

func TestSelectDetailsTimesOutAHangingService(t *testing.T) {
	old := detailTimeout
	detailTimeout = 20 * time.Millisecond
	defer func() { detailTimeout = old }()

	s := NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	s.mu.Lock()
	s.discovery.service = hangingCatalog{}
	s.discovery.catalog = desktopmarket.Catalog{Items: []desktopmarket.CatalogItem{{ID: "fixture/plugin", Repo: "Plugin", InstallSpec: "fixture-plugin"}}}
	s.discovery.revision = 1
	s.mu.Unlock()

	s.SelectDetails("fixture/plugin", false)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if d := s.Discovery(); !d.DetailsLoading {
			if !strings.Contains(d.DetailsErr, "deadline") && !strings.Contains(d.DetailsErr, "canceled") {
				t.Fatalf("details error=%q, want the context deadline", d.DetailsErr)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("details stayed loading past its timeout")
}
