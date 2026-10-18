package market

import (
	"sort"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
)

func (s *Store) SetSort(value string) {
	if value != "stars" && value != "newest" {
		value = "catalog"
	}
	s.mu.Lock()
	s.discovery.sort = value
	s.discovery.page = 0
	s.mu.Unlock()
	s.fire()
}

func (s *Store) SetPeriod(value string) {
	if value != "7" && value != "30" {
		value = "all"
	}
	s.mu.Lock()
	s.discovery.period = value
	s.discovery.page = 0
	s.mu.Unlock()
	s.fire()
}

// filterCatalogListings operates on the discovery rows, never the source catalog.
// Apply listing controls before taking the page so counts and order agree.
func filterCatalogListings(rows []desktopmarket.CatalogItem, order, period string, now time.Time) []desktopmarket.CatalogItem {
	days := 0
	switch period {
	case "7":
		days = 7
	case "30":
		days = 30
	}
	if days > 0 {
		cutoff := now.Add(-time.Duration(days) * 24 * time.Hour).UnixMilli()
		filtered := rows[:0]
		for _, item := range rows {
			if catalogListingTime(item.Added) >= cutoff {
				filtered = append(filtered, item)
			}
		}
		rows = filtered
	}
	if order == "stars" || order == "newest" {
		sort.Slice(rows, func(i, j int) bool {
			a, b := rows[i], rows[j]
			if order == "stars" {
				if a.Stars != b.Stars {
					return a.Stars > b.Stars
				}
			} else if at, bt := catalogListingTime(a.Added), catalogListingTime(b.Added); at != bt {
				return at > bt
			}
			return a.ID < b.ID
		})
	}
	return rows
}

func catalogListingTime(value string) int64 {
	for _, layout := range []string{time.DateOnly, time.RFC3339Nano} {
		if at, err := time.Parse(layout, value); err == nil {
			return at.UnixMilli()
		}
	}
	return 0
}
