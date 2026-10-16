package market

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
)

func TestDiscoverySortAndListingPeriod(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	items := []desktopmarket.CatalogItem{
		{ID: "z/old", Stars: 99, Added: "2026-09-01"},
		{ID: "b/recent", Stars: 10, Added: "2026-10-15"},
		{ID: "a/recent", Stars: 10, Added: "2026-10-15"},
		{ID: "c/week", Stars: 3, Added: "2026-10-09T12:00:00Z"},
		{ID: "d/month", Stars: 5, Added: "2026-09-16T14:00:00+02:00"},
		{ID: "e/invalid", Stars: 20, Added: "not-a-date"},
		{ID: "f/missing", Stars: 0},
	}
	ids := func(rows []desktopmarket.CatalogItem) []string {
		out := make([]string, len(rows))
		for i, row := range rows {
			out[i] = row.ID
		}
		return out
	}
	for _, tc := range []struct {
		name, sort, period string
		want               []string
	}{
		{"catalog", "catalog", "all", []string{"z/old", "b/recent", "a/recent", "c/week", "d/month", "e/invalid", "f/missing"}},
		{"stars", "stars", "all", []string{"z/old", "e/invalid", "a/recent", "b/recent", "d/month", "c/week", "f/missing"}},
		{"newest", "newest", "all", []string{"a/recent", "b/recent", "c/week", "d/month", "z/old", "e/invalid", "f/missing"}},
		{"week-inclusive", "catalog", "7", []string{"b/recent", "a/recent", "c/week"}},
		{"month-inclusive", "newest", "30", []string{"a/recent", "b/recent", "c/week", "d/month"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := filterCatalogListings(append([]desktopmarket.CatalogItem(nil), items...), tc.sort, tc.period, now)
			if got := ids(rows); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("rows = %v, want %v", got, tc.want)
			}
		})
	}
	t.Run("store-and-pagination", func(t *testing.T) {
		s := NewWithPersonalPath("")
		notifications := 0
		s.SetNotify(func() { notifications++ })
		date := time.Now().Add(-24 * time.Hour).UTC().Format(time.DateOnly)
		for i := 0; i < 65; i++ {
			s.discovery.catalog.Items = append(s.discovery.catalog.Items, desktopmarket.CatalogItem{ID: fmt.Sprintf("owner/%02d", i), Stars: float64(i), Added: date, Category: "tools", Description: "searchable"})
		}
		s.discovery.catalog.Items = append(s.discovery.catalog.Items, desktopmarket.CatalogItem{ID: "owner/old", Stars: 100, Added: "2000-01-01", Category: "tools", Description: "searchable"})
		original := append([]desktopmarket.CatalogItem(nil), s.discovery.catalog.Items...)
		if d := s.Discovery(); d.Sort != "catalog" || d.Period != "all" || d.Rows[0].ID != "owner/00" {
			t.Fatalf("unexpected defaults: %+v", d)
		}
		s.SetPage(1)
		s.SetSort("stars")
		if d := s.Discovery(); d.Sort != "stars" || d.Page != 0 || d.Total != 66 || len(d.Rows) != 60 || d.Rows[0].ID != "owner/old" {
			t.Fatalf("sort did not apply before paging: %+v", d)
		}
		s.SetPage(1)
		s.SetPeriod("7")
		if d := s.Discovery(); d.Period != "7" || d.Page != 0 || d.Total != 65 || len(d.Rows) != 60 || d.Rows[0].ID != "owner/64" {
			t.Fatalf("period did not filter before paging/reset page: %+v", d)
		}
		s.SetPage(1)
		if d := s.Discovery(); d.Page != 1 || len(d.Rows) != 5 || d.Rows[0].ID != "owner/04" {
			t.Fatalf("unexpected second page: %+v", d)
		}
		s.SetSort("catalog")
		s.SetPeriod("all")
		if d := s.Discovery(); d.Total != 66 || d.Rows[0].ID != "owner/00" || !reflect.DeepEqual(d.Catalog.Items, original) {
			t.Fatal("restoring choices did not preserve original catalog order")
		}
		s.personal.data.Favorites = map[string]bool{"owner/02": true, "owner/old": true}
		s.SetTab("favorites")
		s.SetCategory("tools")
		s.SetQuery(" SEARCHABLE ")
		s.SetSort("newest")
		s.SetPeriod("30")
		s.discovery.selected = "owner/old"
		if d := s.Discovery(); d.Total != 1 || d.Rows[0].ID != "owner/02" || d.Selected == nil || d.Selected.ID != "owner/old" {
			t.Fatalf("combined filters changed favorites/details: %+v", d)
		}
		s.SetCategory("other")
		if d := s.Discovery(); d.Total != 0 || len(d.Rows) != 0 {
			t.Fatal("category filter was bypassed")
		}
		if notifications < 10 {
			t.Fatal("choice changes did not notify the view")
		}
	})
}
